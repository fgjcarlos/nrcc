package server

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
	"github.com/dop251/goja/token"
	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/fgjcarlos/nrcc/internal/service"
)

func serverApplyResolver(detect func() model.HostStatus, processManager func() *service.ProcessManager, systemd service.SystemdManager, docker *service.DockerService) service.ApplyAdapterResolver {
	return func(req service.ApplyRequest) (service.ApplyAdapter, error) {
		status := detect()
		nodeRed := status.NodeRed
		pm := processManager()
		var readinessResolver func() (string, error)
		switch {
		case pm != nil && !pm.IsExternalMode():
			_, err := applyCandidatePort(req)
			if err != nil {
				return nil, err
			}
			// Authority remains pinned to this ProcessManager; each Ready call
			// reads the active settings path so recovery probes restored config.
			nodeRed = model.NodeRedEnvironment{Detected: true, Mode: model.InstallationModeNative, ManagedByNRCC: true, Running: true}
			readinessResolver = func() (string, error) { return managedSettingsReadinessURL(req.Path, pm) }
		case nodeRed.Mode == model.InstallationModeNative && nodeRed.ManagedByNRCC:
			if _, err := applyCandidatePort(req); err != nil {
				return nil, err
			}
			readinessResolver = func() (string, error) { return managedSettingsReadinessURL(req.Path, pm) }
		case nodeRed.Mode == model.InstallationModeDocker:
			port, err := applyCandidatePort(req)
			if err != nil {
				return nil, err
			}
			// Detection binds readiness to a published mapping for container port
			// 1880. Do not restart if the candidate would move Node-RED elsewhere.
			if port != 1880 {
				return nil, fmt.Errorf("apply: Docker target publishes container port 1880, candidate port %d is unsupported", port)
			}
		case nodeRed.Mode == model.InstallationModeNative && !nodeRed.ManagedByNRCC:
			// Issue #918: bind the systemd readiness probe to the port the
			// candidate settings define instead of a well-known default a
			// foreign listener could occupy. When the strict single-statement
			// parser cannot prove the uiPort static (stock Node-RED templates
			// use `process.env.PORT || 1880` alongside require() calls), fall
			// back to the declared numeric literal so applies that succeeded
			// at baseline keep working; only refuse when no port authority can
			// be derived at all.
			_, portErr := applyCandidatePort(req)
			fallbackPort, hasFallback := fallbackLiteralPort(req)
			if portErr != nil && !hasFallback {
				return nil, portErr
			}
			readinessResolver = func() (string, error) {
				if portErr == nil {
					return settingsReadinessURL(req.Path)
				}
				return fmt.Sprintf("http://127.0.0.1:%d/settings", fallbackPort), nil
			}
		}
		return service.SelectApplyAdapter(req.Capabilities, nodeRed, service.ApplyAdapterDeps{
			ProcessManager: pm, SystemdManager: systemd, Docker: docker,
			ReadinessURLResolver: readinessResolver,
		})
	}
}

var errSettingsPortMissing = errors.New("apply: settings must define exactly one static uiPort")

// fallbackPortRe extracts the declared numeric uiPort literal from raw
// settings the strict parser rejects, e.g. `uiPort: process.env.PORT || 1880`
// (review correction for issue #918: stock templates must keep applying).
var fallbackPortRe = regexp.MustCompile(`uiPort\s*:\s*(?:[^\n,]+?\|\|\s*)?(\d{1,5})\b`)

// fallbackLiteralPort reports the declared fallback uiPort literal of the
// candidate settings, if any.
func fallbackLiteralPort(req service.ApplyRequest) (int, bool) {
	if req.Content == "" {
		return 0, false
	}
	m := fallbackPortRe.FindStringSubmatch(req.Content)
	if m == nil {
		return 0, false
	}
	port, err := strconv.Atoi(m[1])
	if err != nil || port < 1 || port > 65535 {
		return 0, false
	}
	return port, true
}

func managedSettingsReadinessURL(path string, pm *service.ProcessManager) (string, error) {
	if pm == nil {
		return settingsReadinessURL(path)
	}
	// The ProcessManager records the effective --port at Start time from its
	// merged OS, stored-variable, and .env layers. Use that launch authority;
	// do not inspect `/proc` or re-read env files that may have changed since.
	if port := pm.EffectiveRuntimePort(); port > 0 {
		return settingsReadinessURL(path, port)
	}
	return settingsReadinessURL(path)
}

func settingsReadinessURL(path string, runtimePort ...int) (string, error) {
	if len(runtimePort) > 0 && (runtimePort[0] < 1 || runtimePort[0] > 65535) {
		return "", fmt.Errorf("apply: managed Node-RED runtime port %d is outside 1-65535", runtimePort[0])
	}
	// #nosec G304 -- path is the active settings.js path inside the operator-supplied dataDir; not request-derived.
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("apply: read active settings for readiness: %w", err)
	}
	port, err := staticSettingsPort(string(content))
	if err != nil {
		if !errors.Is(err, errSettingsPortMissing) {
			return "", err
		}
		if len(runtimePort) == 0 {
			return "", err
		}
		port = runtimePort[0]
		// A missing setting is safe only when ProcessManager supplies the
		// effective port used for the live runtime. Never guess a listener.

	} else if len(runtimePort) > 0 {
		// CLI --port is the effective listener override, including NODE_RED_PORT.
		port = runtimePort[0]
	}
	return fmt.Sprintf("http://127.0.0.1:%d/settings", port), nil
}

func applyCandidatePort(req service.ApplyRequest) (int, error) {
	if req.StructuredConfig != nil {
		port := req.StructuredConfig.Port
		if port < 1 || port > 65535 {
			return 0, fmt.Errorf("apply: structured candidate port %d is outside 1-65535", port)
		}
		return port, nil
	}
	return staticSettingsPort(req.Content)
}

// staticSettingsPort accepts only an unambiguous static module.exports object
// with one numeric uiPort literal. Parsing never evaluates candidate JavaScript.
func staticSettingsPort(content string) (int, error) {
	program, err := parser.ParseFile(nil, "settings.js", content, 0)
	if err != nil {
		// The operator-facing contract for invalid raw settings is a syntax
		// error (asserted by frontend/e2e/stack.spec.ts), so keep that phrase.
		return 0, fmt.Errorf("apply: candidate settings syntax error: %w", err)
	}
	if len(program.Body) != 1 {
		return 0, fmt.Errorf("apply: settings readiness port is ambiguous")
	}
	statement, ok := program.Body[0].(*ast.ExpressionStatement)
	if !ok {
		return 0, fmt.Errorf("apply: settings readiness port is ambiguous")
	}
	assignment, ok := statement.Expression.(*ast.AssignExpression)
	if !ok || assignment.Operator != token.ASSIGN {
		return 0, fmt.Errorf("apply: settings readiness port is ambiguous")
	}
	left, ok := assignment.Left.(*ast.DotExpression)
	if !ok || identifierName(left.Left) != "module" || left.Identifier.Name.String() != "exports" {
		return 0, fmt.Errorf("apply: settings readiness port is ambiguous")
	}
	object, ok := assignment.Right.(*ast.ObjectLiteral)
	if !ok {
		return 0, fmt.Errorf("apply: settings readiness port is ambiguous")
	}
	port, count := 0, 0
	for _, property := range object.Value {
		if _, spread := property.(*ast.SpreadElement); spread {
			return 0, fmt.Errorf("apply: settings object spread makes readiness port ambiguous")
		}
		keyed, ok := property.(*ast.PropertyKeyed)
		if !ok {
			continue
		}
		if keyed.Computed {
			return 0, fmt.Errorf("apply: computed settings property makes readiness port ambiguous")
		}
		name := propertyName(keyed.Key)
		if name != "uiPort" {
			continue
		}
		count++
		if keyed.Kind != ast.PropertyKindValue {
			return 0, fmt.Errorf("apply: settings uiPort is not a static property")
		}
		number, ok := keyed.Value.(*ast.NumberLiteral)
		if !ok {
			return 0, fmt.Errorf("apply: settings uiPort is not a numeric literal")
		}
		parsed, parseErr := strconv.ParseInt(number.Literal, 0, 64)
		if parseErr != nil || parsed < 1 || parsed > 65535 {
			return 0, fmt.Errorf("apply: settings uiPort literal is outside 1-65535")
		}
		port = int(parsed)
	}
	if count == 0 {
		return 0, errSettingsPortMissing
	}
	if count != 1 {
		return 0, fmt.Errorf("apply: settings must define exactly one static uiPort")
	}
	return port, nil
}

func identifierName(expr ast.Expression) string {
	identifier, ok := expr.(*ast.Identifier)
	if !ok {
		return ""
	}
	return identifier.Name.String()
}

func propertyName(expr ast.Expression) string {
	switch key := expr.(type) {
	case *ast.Identifier:
		return key.Name.String()
	case *ast.StringLiteral:
		return key.Value.String()
	default:
		return ""
	}
}
