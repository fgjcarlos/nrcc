package server

import (
	"fmt"
	"os"
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
		if pm != nil && !pm.IsExternalMode() {
			_, err := applyCandidatePort(req)
			if err != nil {
				return nil, err
			}
			// Authority remains pinned to this ProcessManager; each Ready call
			// reads the active settings path so recovery probes restored config.
			nodeRed = model.NodeRedEnvironment{Detected: true, Mode: model.InstallationModeNative, ManagedByNRCC: true, Running: true}
			readinessResolver = func() (string, error) { return settingsReadinessURL(req.Path) }
		} else if nodeRed.Mode == model.InstallationModeNative && nodeRed.ManagedByNRCC {
			if _, err := applyCandidatePort(req); err != nil {
				return nil, err
			}
			readinessResolver = func() (string, error) { return settingsReadinessURL(req.Path) }
		} else if nodeRed.Mode == model.InstallationModeDocker {
			port, err := applyCandidatePort(req)
			if err != nil {
				return nil, err
			}
			// Detection binds readiness to a published mapping for container port
			// 1880. Do not restart if the candidate would move Node-RED elsewhere.
			if port != 1880 {
				return nil, fmt.Errorf("apply: Docker target publishes container port 1880, candidate port %d is unsupported", port)
			}
		}
		return service.SelectApplyAdapter(req.Capabilities, nodeRed, service.ApplyAdapterDeps{
			ProcessManager: pm, SystemdManager: systemd, Docker: docker,
			ReadinessURLResolver: readinessResolver,
		})
	}
}

func settingsReadinessURL(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("apply: read active settings for readiness: %w", err)
	}
	port, err := staticSettingsPort(string(content))
	if err != nil {
		return "", err
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
		return 0, fmt.Errorf("apply: cannot parse candidate settings for readiness: %w", err)
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
		name := propertyName(keyed.Key)
		if name != "uiPort" {
			continue
		}
		count++
		if keyed.Kind != ast.PropertyKindValue || keyed.Computed {
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
