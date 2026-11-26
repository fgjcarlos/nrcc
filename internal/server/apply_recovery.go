package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

// RecoverExternalApplyRuntime restarts the detected external runtime after boot restores settings.js.
func RecoverExternalApplyRuntime(ctx context.Context, host *service.HostService, config *service.ConfigService, dataDir, settingsPath, restored string) error {
	status := host.Detect()
	identity := runtimeIdentity(status.NodeRed)
	if status.Settings.Path != settingsPath {
		return fmt.Errorf("runtime %s settings path changed to %q; runtime may still run candidate settings", identity, status.Settings.Path)
	}
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("runtime %s may still run candidate settings: read restored settings: %w", identity, err)
	}
	resolve := serverApplyResolver(func() model.HostStatus { return status }, func() *service.ProcessManager { return nil }, service.NewSystemdManager(), service.NewDockerService())
	return recoverExternalApply(ctx, status, config, dataDir, settingsPath, restored, string(content), resolve)
}

func recoverExternalApply(ctx context.Context, status model.HostStatus, config *service.ConfigService, dataDir, settingsPath, restored, content string, resolve service.ApplyAdapterResolver) error {
	identity := runtimeIdentity(status.NodeRed)
	adapter, err := resolve(service.ApplyRequest{Path: settingsPath, Content: content, Capabilities: config.ConfigurationCapabilities()})
	if err != nil {
		return fmt.Errorf("runtime %s may still run candidate settings (restored snapshot %s): resolve adapter: %w", identity, restored, err)
	}
	audit := initAuditService(dataDir, logStartupWarning)
	var hook service.AuditHookFunc
	if audit != nil {
		hook = func(r *http.Request, actor, action, target, result string, meta map[string]string) {
			audit.Log(r, actor, action, target, result, meta)
		}
	}
	err = service.RecoverExternalApplyRuntime(ctx, service.RollbackRequest{
		BackupPath: restored, ActivePath: settingsPath, Stage: service.RollbackStage("boot"), Adapter: adapter,
		Request: &http.Request{}, Actor: "system", AuditHook: hook, Capabilities: config.ConfigurationCapabilities(),
	})
	if err != nil {
		return fmt.Errorf("runtime %s may still run candidate settings (restored snapshot %s): %w", identity, restored, err)
	}
	return nil
}

func runtimeIdentity(runtime model.NodeRedEnvironment) string {
	if runtime.ContainerName != "" {
		return runtime.ContainerName
	}
	if runtime.ContainerID != "" {
		return runtime.ContainerID
	}
	if runtime.Executable != "" {
		return runtime.Executable
	}
	if runtime.Mode != model.InstallationModeNone {
		return string(runtime.Mode)
	}
	return "unknown Node-RED runtime"
}

func logStartupWarning(format string, args ...any) { log.Printf(format, args...) }
