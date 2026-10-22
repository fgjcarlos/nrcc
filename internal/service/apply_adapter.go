// Package service — slice B of issue #758 delivers the runtime adapters
// that restart Node-RED after a successful atomic settings.js write and
// verify it came back healthy. Slice A introduced the apply pipeline
// (validate → backup → atomic write → audit); slice B adds the post-write
// stages (Restart → Ready) plus a rollback path that restores the
// snapshot from slice A's backup when either stage fails.
//
// Adapter contract
// ================
//
// ApplyAdapter is selected ONCE per apply transaction by
// SelectApplyAdapter from (ConfigurationCapabilities.Adapter,
// NodeRedEnvironment.Mode) and is reused for both Restart and Ready so
// the readiness probe targets the same Node-RED that was just
// restarted. Re-resolving during readiness would let a misbehaving
// selector flip from "native-binary" to "docker-compose" mid-flight,
// silently probing a different process than the one we restarted.
//
//   - Restart blocks until the restart signal is delivered. It does
//     NOT block on readiness — that is the Ready stage's job — so a
//     slow Node-RED boot doesn't pin the apply goroutine forever.
//   - Ready probes the Node-RED admin HTTP endpoint with bounded
//     retries. A successful probe (HTTP 2xx/401/403) means Node-RED
//     accepted the new settings; a 503 or connection-refused means
//     it did not and the orchestrator should roll back.
//
// The four implementations map onto the slice A inputs:
//
//	native-binary   ← Mode=Native, ManagedByNRCC=true  (ProcessManager)
//	native-systemd  ← Mode=Native, ManagedByNRCC=false (systemd unit)
//	docker-compose  ← Mode=Docker, ManagedByNRCC=true  (compose project)
//	docker-custom   ← Mode=Docker, ManagedByNRCC=false (loose container)

package service

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// ApplyAdapterName values are the canonical legacy identifiers surfaced in
// audit meta. Keep them stable: the observability dashboard filters restart
// events by adapter name.
const dockerRestartTimeout = 30 * time.Second

const (
	AdapterNameNativeBinary  = "native-binary"
	AdapterNameNativeSystemd = "native-systemd"
	AdapterNameDockerCompose = "docker-compose"
	AdapterNameDockerCustom  = "docker-custom"
)

// ErrUnsupportedAdapter is returned by SelectApplyAdapter when the
// runtime environment does not map onto any of the four ApplyAdapter
// implementations. Callers surface it as a 409 with code
// CONFIG_ADAPTER_UNSUPPORTED in slice C.
var ErrUnsupportedAdapter = errors.New("apply: no adapter matches the current Node-RED environment")

// ErrAdapterUnitMissing is returned by nativeSystemdAdapter.Restart
// when the systemd unit name (carried in NodeRedEnvironment.ContainerName)
// is empty. A native install without a unit name cannot be restarted
// by the adapter and the orchestrator must roll back.
var ErrAdapterUnitMissing = errors.New("apply: systemd unit name is empty")

// ApplyAdapter is the per-environment restart + readiness primitive.
//
// Implementations are expected to be cheap to construct (no I/O in the
// constructor); the heavy work happens in Restart and Ready which the
// orchestrator invokes after a successful atomic write.
type ApplyAdapter interface {
	// Name returns the adapter's stable canonical identifier. It is the
	// audit identity fallback for adapters without an optional AuditName
	// method; operation details may be provided separately.
	Name() string

	// Restart issues the platform-specific restart signal. It returns
	// nil once the signal has been delivered (not once Node-RED is
	// ready). A non-nil error means the signal failed; the caller
	// should treat that as a hard failure and trigger rollback.
	//
	// Restart honours ctx cancellation BEFORE issuing the signal so
	// a cancelled apply doesn't leave a half-restarted Node-RED.
	Restart(ctx context.Context) error

	// Ready probes Node-RED's HTTP admin endpoint until it responds.
	// The readiness contract is:
	//   - HTTP 200/204/301/302/401/403 → ready (Node-RED is up).
	//   - HTTP 503 or connection refused → not ready.
	//   - ctx cancellation → ctx.Err().
	Ready(ctx context.Context) error
}

// ApplyAdapterDeps groups the optional restart primitives the
// selector needs. Tests construct a value with only the fields they
// exercise; production callers wire every dependency so any of the
// four adapters can be resolved.
//
// ProcessManager / SystemdManager / DockerService are nullable because
// not every deployment exposes all three (e.g. a native systemd
// install does not need DockerService). ExecCommand defaults to
// exec.Command via the package-level execCommand shim.
type ApplyAdapterDeps struct {
	ProcessManager       *ProcessManager
	SystemdManager       SystemdManager
	Docker               *DockerService
	ExecCommand          func(name string, arg ...string) *exec.Cmd
	ReadinessURLResolver func() (string, error)
}

// SelectApplyAdapter chooses the right ApplyAdapter for the runtime
// detected by HostService.Detect. The decision is made ONCE per apply
// and captured in the audit record; subsequent stages reuse the same
// adapter instance so the readiness probe targets the same Node-RED
// that was just restarted.
//
// Selection matrix:
//
//	Mode=Native, ManagedByNRCC=true   → native-binary
//	Mode=Native, ManagedByNRCC=false  → native-systemd
//	Mode=Docker, ManagedByNRCC=true   → pinned docker target
//	Mode=Docker, ManagedByNRCC=false  → pinned docker target
//	Anything else / Adapter!="nodered-5" → ErrUnsupportedAdapter
//
// capabilities is the ConfigurationCapabilities from slice A's
// inputs and nodeRed carries the install authority (Mode,
// ManagedByNRCC, ContainerName). For non-managed native installs
// the systemd unit name is carried in nodeRed.ContainerName; for
// managed docker installs it may contain the project name.
func SelectApplyAdapter(capabilities model.ConfigurationCapabilities, nodeRed model.NodeRedEnvironment, deps ApplyAdapterDeps) (ApplyAdapter, error) {
	// Slice A's compatibility policy gates editing on Adapter ==
	// "nodered-5". The apply pipeline is a no-op against read-only
	// adapters (a 4 read-only adapter would still let restart run,
	// but the orchestrator refuses to start one in the first place).
	if capabilities.Adapter != "nodered-5" {
		return nil, fmt.Errorf("%w: capabilities.adapter=%q", ErrUnsupportedAdapter, capabilities.Adapter)
	}
	if nodeRed.Ambiguous {
		return nil, fmt.Errorf("%w: Node-RED authority is ambiguous", ErrUnsupportedAdapter)
	}
	switch nodeRed.Mode {
	case model.InstallationModeNative:
		if nodeRed.ManagedByNRCC {
			if deps.ProcessManager == nil {
				return nil, fmt.Errorf("%w: ProcessManager dependency not wired", ErrUnsupportedAdapter)
			}
			readinessResolver := deps.ReadinessURLResolver
			if readinessResolver == nil && strings.TrimSpace(nodeRed.ReadinessURL) != "" {
				readinessResolver = func() (string, error) { return nodeRed.ReadinessURL, nil }
			}
			if readinessResolver == nil {
				return nil, fmt.Errorf("%w: managed native readiness endpoint is unknown", ErrUnsupportedAdapter)
			}
			return &nativeBinaryAdapter{pm: deps.ProcessManager, readinessURLResolver: readinessResolver}, nil
		}
		if deps.SystemdManager == nil {
			return nil, fmt.Errorf("%w: SystemdManager dependency not wired", ErrUnsupportedAdapter)
		}
		if strings.TrimSpace(nodeRed.ContainerName) == "" {
			return nil, fmt.Errorf("%w: systemd unit name is empty", ErrUnsupportedAdapter)
		}
		return &nativeSystemdAdapter{
			mgr:         deps.SystemdManager,
			unit:        nodeRed.ContainerName,
			execCommand: ensureExecCommand(deps.ExecCommand),
		}, nil
	case model.InstallationModeDocker:
		if strings.TrimSpace(nodeRed.ContainerID) == "" || strings.TrimSpace(nodeRed.ReadinessURL) == "" {
			return nil, fmt.Errorf("%w: Docker target ID and published readiness endpoint are required", ErrUnsupportedAdapter)
		}
		auditName := AdapterNameDockerCustom
		if nodeRed.ManagedByNRCC {
			auditName = AdapterNameDockerCompose
		}
		return &dockerTargetAdapter{
			execCommand:  ensureExecCommand(deps.ExecCommand),
			containerID:  strings.TrimSpace(nodeRed.ContainerID),
			readinessURL: strings.TrimSpace(nodeRed.ReadinessURL),
			auditName:    auditName,
		}, nil
	default:
		return nil, fmt.Errorf("%w: nodeRed.mode=%q", ErrUnsupportedAdapter, nodeRed.Mode)
	}
}

// ensureExecCommand returns deps.ExecCommand if non-nil, otherwise the
// package-level execCommand shim (which defaults to exec.Command).
func ensureExecCommand(deps func(name string, arg ...string) *exec.Cmd) func(name string, arg ...string) *exec.Cmd {
	if deps != nil {
		return deps
	}
	return execCommand
}

// nativeBinaryAdapter restarts Node-RED via the in-process ProcessManager.
// This is the path used when NRCC itself spawned the child process and
// holds the lifecycle authority (the managed-runtime detection in
// model.NodeRedEnvironment.ManagedByNRCC).
type nativeBinaryAdapter struct {
	pm                   *ProcessManager
	readinessURLResolver func() (string, error)
}

func (a *nativeBinaryAdapter) Name() string { return AdapterNameNativeBinary }

func (a *nativeBinaryAdapter) Restart(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.pm == nil {
		return fmt.Errorf("%w: ProcessManager is nil", ErrUnsupportedAdapter)
	}
	return a.pm.Restart()
}

func (a *nativeBinaryAdapter) Ready(ctx context.Context) error {
	url, err := a.readinessURLResolver()
	if err != nil {
		return err
	}
	return probeAdminReady(ctx, url)
}

// nativeSystemdAdapter restarts Node-RED via `systemctl restart <unit>`.
// The unit name is captured from NodeRedEnvironment.ContainerName
// (slice A repurposes the field as the systemd unit identifier for
// non-managed native installations; the field is reserved for that
// purpose in the same way managed docker installs reuse it for the
// compose project name).
type nativeSystemdAdapter struct {
	mgr         SystemdManager
	unit        string
	execCommand func(name string, arg ...string) *exec.Cmd
}

func (a *nativeSystemdAdapter) Name() string { return AdapterNameNativeSystemd }

func (a *nativeSystemdAdapter) Restart(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.unit == "" {
		return ErrAdapterUnitMissing
	}
	// We intentionally do NOT extend the SystemdManager interface
	// (Stop/Start are the existing primitives). `systemctl restart`
	// is a single atomic operation and keeps the failure surface
	// narrow — Stop+Start could leave a half-restarted unit if Start
	// fails after Stop succeeded. The exec shim is the same one the
	// package uses everywhere else so tests can stub it.
	// #nosec G204 -- unit is the operator-supplied systemd unit name from NodeRedEnvironment, not request-derived.
	cmd := a.execCommand("systemctl", "restart", a.unit)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl restart %s: %w", a.unit, err)
	}
	return nil
}

func (a *nativeSystemdAdapter) Ready(ctx context.Context) error {
	return probeAdminReady(ctx, nativeDefaultAdminURL())
}

// dockerComposeAdapter restarts Node-RED via `docker compose restart`.
// The compose project name comes from NodeRedEnvironment.ContainerName
// (the field slice A already populates for managed docker installs).
type dockerComposeAdapter struct {
	execCommand func(name string, arg ...string) *exec.Cmd
	projectName string
}

type dockerTargetAdapter struct {
	execCommand  func(name string, arg ...string) *exec.Cmd
	containerID  string
	readinessURL string
	auditName    string
}

func (a *dockerComposeAdapter) Name() string { return AdapterNameDockerCompose }

func (a *dockerComposeAdapter) Restart(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	args := []string{"compose", "restart"}
	if a.projectName != "" {
		// `-p` pins the project so a multi-project host cannot
		// accidentally restart a sibling service. We still tolerate
		// the empty-project case for single-project hosts.
		args = append(args, "-p", a.projectName)
	}
	// #nosec G204 -- args are composed from the operator-controlled project name and a fixed subcommand list; not request-derived.
	cmd := a.execCommand("docker", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose restart: %w", err)
	}
	return nil
}

func (a *dockerComposeAdapter) Ready(ctx context.Context) error {
	return probeAdminReady(ctx, nativeDefaultAdminURL())
}

// dockerCustomAdapter restarts Node-RED via DockerService.Restart,
// which runs `docker restart <container-name>` against the discovered
// Node-RED container. This is the path for native docker installs
// where NRCC did not bring up the container.
type dockerCustomAdapter struct {
	docker *DockerService
}

func (a *dockerCustomAdapter) Name() string { return AdapterNameDockerCustom }

func (a *dockerCustomAdapter) Restart(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.docker == nil {
		return fmt.Errorf("%w: DockerService is nil", ErrUnsupportedAdapter)
	}
	return a.docker.Restart()
}

func (a *dockerCustomAdapter) Ready(ctx context.Context) error {
	return probeAdminReady(ctx, nativeDefaultAdminURL())
}

func (a *dockerTargetAdapter) Name() string { return AdapterNameDockerCustom }

// AuditName preserves the historical topology label used by dashboard filters.
func (a *dockerTargetAdapter) AuditName() string {
	if a.auditName != "" {
		return a.auditName
	}
	return a.Name()
}

// OperationName describes the command actually used for the pinned target.
func (a *dockerTargetAdapter) OperationName() string { return "docker restart" }

func (a *dockerTargetAdapter) Restart(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.containerID == "" {
		return fmt.Errorf("%w: Docker container ID is empty", ErrUnsupportedAdapter)
	}
	boundedCtx := ctx
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok {
		boundedCtx, cancel = context.WithTimeout(ctx, dockerRestartTimeout)
	}
	defer cancel()
	cmd := a.execCommand("docker", "restart", a.containerID)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("docker restart %s: %w", a.containerID, err)
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	select {
	case err := <-wait:
		if err != nil {
			return fmt.Errorf("docker restart %s: %w", a.containerID, err)
		}
		return nil
	case <-boundedCtx.Done():
		_ = cmd.Process.Kill()
		<-wait
		return fmt.Errorf("docker restart %s: %w", a.containerID, boundedCtx.Err())
	}
}

func (a *dockerTargetAdapter) Ready(ctx context.Context) error {
	return probeAdminReady(ctx, a.readinessURL)
}

func applyAuditName(adapter ApplyAdapter) string {
	if identified, ok := adapter.(interface{ AuditName() string }); ok {
		return identified.AuditName()
	}
	return adapter.Name()
}

func applyOperationName(adapter ApplyAdapter) string {
	if identified, ok := adapter.(interface{ OperationName() string }); ok {
		return identified.OperationName()
	}
	return adapter.Name()
}

// nativeDefaultAdminURL returns the canonical Node-RED admin probe URL
// for non-container adapters. Slice C may override the URL via a
// per-deploy override, but the slice B default keeps the probe
// reachable on the well-known port 1880 /settings endpoint.
func nativeDefaultAdminURL() string {
	return "http://127.0.0.1:1880/settings"
}
