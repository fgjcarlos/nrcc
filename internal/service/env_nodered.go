package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fgjcarlos/nrcc/internal/model"
	"github.com/google/uuid"
)

// EnvService handles environment variable operations
type nodeRedGlobalEnv struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

func nodeRedGlobalEnvType(typ string) string {
	switch typ {
	case "number":
		return "num"
	case "boolean":
		return "bool"
	default:
		return "str"
	}
}

// activeNodeRedUserDir resolves the directory that Node-RED uses for the
// flow file. The contract mirrors ProcessManager.resolveNodeRedRuntime so
// env sync follows overrides such as NODE_RED_USER_DIR and NODE_RED_SETTINGS.
func (s *EnvService) activeNodeRedUserDir() (string, error) {
	if s == nil || s.configSvc == nil {
		return "", fmt.Errorf("env service not initialised")
	}
	base := s.configSvc.dataDir
	envMap := map[string]string{}
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if stored, err := s.GetAll(); err == nil {
		for k, v := range stored {
			envMap[k] = v
		}
	}
	dotenvPath := filepath.Join(base, ".env")
	if dotenvVars, err := parseEnvFile(dotenvPath); err == nil {
		for k, v := range dotenvVars {
			envMap[k] = v
		}
	}
	rt, err := resolveNodeRedRuntime(envMap, base)
	if err != nil {
		return "", fmt.Errorf("resolve Node-RED runtime: %w", err)
	}
	return rt.UserDir, nil
}

// syncNodeRedGlobalEnv updates the Node-RED 5 global-config node stored in
// flows.json. A nil variable removes the key; secrets call this path with nil
// because their values must remain in NRCC's encrypted store and process env.
func (s *EnvService) syncNodeRedGlobalEnv(key string, envVar *model.EnvVar) error {
	if err := ValidateEnvKey(key); err != nil {
		return err
	}
	cfg, err := s.configSvc.Get()
	if err != nil {
		return fmt.Errorf("read Node-RED config: %w", err)
	}

	flowFile := cfg.FlowFile
	if flowFile == "" {
		flowFile = "flows.json"
	}
	flowDir, err := s.activeNodeRedUserDir()
	if err != nil {
		return err
	}
	flowPath := filepath.Join(flowDir, flowFile)
	flowPath = filepath.Clean(flowPath)
	dataRoot, err := filepath.Abs(s.configSvc.dataDir)
	if err != nil {
		return fmt.Errorf("resolve data directory: %w", err)
	}
	flowPath, err = filepath.Abs(flowPath)
	if err != nil {
		return fmt.Errorf("resolve flow file: %w", err)
	}
	rel, err := filepath.Rel(dataRoot, flowPath)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || (len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator)) {
		return fmt.Errorf("flow file must stay inside Node-RED data directory")
	}

	mode := os.FileMode(0o644)
	data, err := os.ReadFile(flowPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read Node-RED flows: %w", err)
		}
		data = []byte("[]")
	} else if info, statErr := os.Stat(flowPath); statErr == nil {
		mode = info.Mode().Perm()
	}

	var flows []map[string]json.RawMessage
	if err := json.Unmarshal(data, &flows); err != nil {
		return fmt.Errorf("parse Node-RED flows: %w", err)
	}

	globalIndex := -1
	globalCount := 0
	for i, flow := range flows {
		var typ string
		if raw, ok := flow["type"]; ok && json.Unmarshal(raw, &typ) == nil && typ == "global-config" {
			globalIndex = i
			globalCount++
		}
	}
	if globalCount > 1 {
		return fmt.Errorf("multiple Node-RED global-config nodes found")
	}

	if globalIndex == -1 {
		if envVar == nil {
			return nil
		}
		id, _ := json.Marshal(uuid.NewString())
		typ, _ := json.Marshal("global-config")
		modules, _ := json.Marshal(map[string]any{})
		flows = append(flows, map[string]json.RawMessage{
			"id":      id,
			"type":    typ,
			"modules": modules,
		})
		globalIndex = len(flows) - 1
	}

	global := flows[globalIndex]
	var items []map[string]json.RawMessage
	if raw, ok := global["env"]; ok {
		if err := json.Unmarshal(raw, &items); err != nil {
			return fmt.Errorf("parse Node-RED global environment: %w", err)
		}
	}

	updated := make([]map[string]json.RawMessage, 0, len(items)+1)
	for _, item := range items {
		var name string
		if raw, ok := item["name"]; ok {
			_ = json.Unmarshal(raw, &name)
		}
		if name != key {
			updated = append(updated, item)
		}
	}
	if envVar != nil {
		raw, err := json.Marshal(nodeRedGlobalEnv{
			Name:  envVar.Key,
			Value: envVar.Value,
			Type:  nodeRedGlobalEnvType(envVar.Type),
		})
		if err != nil {
			return fmt.Errorf("encode Node-RED global environment variable: %w", err)
		}
		var item map[string]json.RawMessage
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		updated = append(updated, item)
	}

	rawEnv, err := json.Marshal(updated)
	if err != nil {
		return fmt.Errorf("encode Node-RED global environment: %w", err)
	}
	global["env"] = rawEnv

	output, err := json.MarshalIndent(flows, "", "    ")
	if err != nil {
		return fmt.Errorf("encode Node-RED flows: %w", err)
	}
	output = append(output, '\n')
	if bytes.Equal(data, output) {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(flowPath), 0o755); err != nil {
		return fmt.Errorf("create Node-RED flow directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(flowPath), ".flows-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary Node-RED flow file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set temporary Node-RED flow permissions: %w", err)
	}
	if _, err := tmp.Write(output); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary Node-RED flows: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary Node-RED flows: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary Node-RED flows: %w", err)
	}
	if err := os.Rename(tmpPath, flowPath); err != nil {
		return fmt.Errorf("publish Node-RED flows: %w", err)
	}
	return nil
}

// nodeRedGlobalEnvList reads every env entry in flows.json. The result is
// used both to push and to pull, so keep the source of truth single.
func (s *EnvService) nodeRedGlobalEnvList() ([]nodeRedGlobalEnv, error) {
	if s == nil || s.configSvc == nil {
		return nil, fmt.Errorf("env service not initialised")
	}
	cfg, err := s.configSvc.Get()
	if err != nil {
		return nil, fmt.Errorf("read Node-RED config: %w", err)
	}
	flowFile := cfg.FlowFile
	if flowFile == "" {
		flowFile = "flows.json"
	}
	flowDir, err := s.activeNodeRedUserDir()
	if err != nil {
		return nil, err
	}
	flowPath := filepath.Clean(filepath.Join(flowDir, flowFile))
	dataRoot, err := filepath.Abs(s.configSvc.dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}
	flowPath, err = filepath.Abs(flowPath)
	if err != nil {
		return nil, fmt.Errorf("resolve flow file: %w", err)
	}
	rel, err := filepath.Rel(dataRoot, flowPath)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || (len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("flow file must stay inside Node-RED data directory")
	}
	data, err := os.ReadFile(flowPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read Node-RED flows: %w", err)
	}
	var flows []map[string]json.RawMessage
	if err := json.Unmarshal(data, &flows); err != nil {
		return nil, fmt.Errorf("parse Node-RED flows: %w", err)
	}
	var entries []nodeRedGlobalEnv
	for _, item := range flows {
		if !hasNodeRedGlobalEnvType(item) {
			continue
		}
		raw, ok := item["env"]
		if !ok {
			continue
		}
		var envList []nodeRedGlobalEnv
		if err := json.Unmarshal(raw, &envList); err != nil {
			return nil, fmt.Errorf("parse global-config env: %w", err)
		}
		entries = append(entries, envList...)
	}
	return entries, nil
}

func hasNodeRedGlobalEnvType(item map[string]json.RawMessage) bool {
	raw, ok := item["type"]
	if !ok {
		return false
	}
	var typ string
	if err := json.Unmarshal(raw, &typ); err != nil {
		return false
	}
	return typ == "global-config"
}

// ImportFromNodeRed snapshots the Node-RED 5 global-config env entries and
// merges every new key into NRCC. Keys already present are skipped to keep
// the operation idempotent; secrets remain encrypted in NRCC and never
// leave flows.json. commit=false performs a dry run.
func (s *EnvService) ImportFromNodeRed(commit bool, stopAndRestart func(func() error) (bool, error)) (BulkEnvResult, error) {
	entries, err := s.nodeRedGlobalEnvList()
	if err != nil {
		return BulkEnvResult{}, err
	}
	if len(entries) == 0 {
		return BulkEnvResult{
			Lines:   []BulkEnvLine{},
			Issues:  []BulkEnvIssue{},
			Valid:   true,
			Summary: "no global-config entries in Node-RED",
		}, nil
	}
	existing, err := s.List()
	if err != nil {
		return BulkEnvResult{}, err
	}
	managed := make(map[string]struct{}, len(existing))
	for _, ev := range existing {
		managed[ev.Key] = struct{}{}
	}

	var (
		toImport []BulkEnvLine
		issues   []BulkEnvIssue
		seen     = map[string]int{}
	)
	for i, e := range entries {
		line := i + 1
		if e.Name == "" {
			issues = append(issues, BulkEnvIssue{Line: line, Reason: "entry is missing a name"})
			continue
		}
		if _, ok := managed[e.Name]; ok {
			issues = append(issues, BulkEnvIssue{Line: line, Key: e.Name, Reason: "already managed by NRCC"})
			continue
		}
		if _, ok := seen[e.Name]; ok {
			continue
		}
		typ := nodeRedTypeToValueType(e.Type)
		if err := ValidateEnvKey(e.Name); err != nil {
			issues = append(issues, BulkEnvIssue{Line: line, Key: e.Name, Reason: err.Error()})
			continue
		}
		if err := ValidateValue(e.Value, typ); err != nil {
			issues = append(issues, BulkEnvIssue{Line: line, Key: e.Name, Reason: err.Error()})
			continue
		}
		seen[e.Name] = line
		toImport = append(toImport, BulkEnvLine{Line: line, Key: e.Name, Value: e.Value, Type: typ})
	}

	result := BulkEnvResult{
		Lines:  toImport,
		Issues: issues,
		Valid:  len(toImport) > 0,
	}
	switch {
	case len(toImport) == 0 && len(issues) == 0:
		result.Summary = "no new entries to import"
	case len(issues) > 0:
		result.Summary = fmt.Sprintf("%d skipped, %d ready", len(issues), len(toImport))
	default:
		result.Summary = fmt.Sprintf("%d variable(s) ready", len(toImport))
	}
	if !commit || !result.Valid {
		return result, nil
	}
	apply := func() error {
		for _, line := range toImport {
			if err := s.Set(line.Key, line.Value, line.Type, "imported from Node-RED", false); err != nil {
				return fmt.Errorf("line %d (%s): %w", line.Line, line.Key, err)
			}
		}
		return nil
	}
	if stopAndRestart != nil {
		if _, err := stopAndRestart(apply); err != nil {
			return result, err
		}
	} else if err := apply(); err != nil {
		return result, err
	}
	return result, nil
}

func nodeRedTypeToValueType(nodeRedType string) string {
	switch nodeRedType {
	case "num":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

// keep imports referenced when running tests on stripped builds
var _ = time.Now
var _ = strconv.Atoi
var _ = bytes.Equal
var _ bufio.Scanner
