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
	base := s.configSvc.dataDir
	envMap := map[string]string{}
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	if s != nil {
		if stored, err := s.GetAll(); err == nil {
			for k, v := range stored {
				envMap[k] = v
			}
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

// keep imports referenced when running tests on stripped builds
var _ = time.Now
var _ = strconv.Atoi
var _ = bytes.Equal
var _ bufio.Scanner
