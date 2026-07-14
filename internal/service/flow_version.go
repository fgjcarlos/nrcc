package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	versionDir   = "flow-versions"
	maxVersions  = 100
	pollInterval = 10 * time.Second
)

type FlowVersion struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Hash      string `json:"hash"`
	NodeCount int    `json:"nodeCount"`
	Size      int    `json:"size"`
}

type FlowDiff struct {
	Added    []FlowDiffEntry `json:"added,omitempty"`
	Removed  []FlowDiffEntry `json:"removed,omitempty"`
	Modified []FlowDiffMod   `json:"modified,omitempty"`
}

type FlowDiffEntry struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Label string `json:"label,omitempty"`
}

type FlowDiffMod struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Label   string   `json:"label,omitempty"`
	Changed []string `json:"changed"`
}

type FlowVersionService struct {
	dataDir  string
	mu       sync.Mutex
	lastHash string
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewFlowVersionService(dataDir string) *FlowVersionService {
	dir := filepath.Join(dataDir, versionDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		// best-effort; CaptureNow will surface a clearer error later
		_ = err
	}
	return &FlowVersionService{dataDir: dataDir, stopCh: make(chan struct{})}
}

func (s *FlowVersionService) StartPolling() {
	go func() {
		s.captureIfChanged()
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.captureIfChanged()
			case <-s.stopCh:
				return
			}
		}
	}()
}

// Stop signals the polling goroutine to exit. It is safe to call multiple times.
func (s *FlowVersionService) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *FlowVersionService) captureIfChanged() {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "flows.json"))
	if err != nil {
		return
	}

	hash := hashBytes(data)

	s.mu.Lock()
	defer s.mu.Unlock()

	if hash == s.lastHash {
		return
	}
	s.lastHash = hash

	_ = s.saveVersion(data, hash)
}

func (s *FlowVersionService) CaptureNow() error {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "flows.json"))
	if err != nil {
		return fmt.Errorf("read flows: %w", err)
	}

	hash := hashBytes(data)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastHash = hash
	return s.saveVersion(data, hash)
}

func (s *FlowVersionService) saveVersion(data []byte, hash string) error {
	ts := time.Now().UTC().Format("20060102-150405")
	name := fmt.Sprintf("%s_%s.json", ts, hash[:8])
	path := filepath.Join(s.dataDir, versionDir, name)

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	s.pruneOld()
	return nil
}

func (s *FlowVersionService) ListVersions() ([]FlowVersion, error) {
	dir := filepath.Join(s.dataDir, versionDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []FlowVersion{}, nil
		}
		return nil, err
	}

	var versions []FlowVersion
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(e.Name(), ".json")
		parts := strings.SplitN(name, "_", 2)

		ts := ""
		hash := ""
		if len(parts) == 2 {
			if t, err := time.Parse("20060102-150405", parts[0]); err == nil {
				ts = t.Format(time.RFC3339)
			}
			hash = parts[1]
		}

		nodeCount := countNodes(filepath.Join(dir, e.Name()))

		versions = append(versions, FlowVersion{
			ID:        e.Name(),
			Timestamp: ts,
			Hash:      hash,
			NodeCount: nodeCount,
			Size:      int(info.Size()),
		})
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].ID > versions[j].ID
	})

	return versions, nil
}

func (s *FlowVersionService) GetVersion(id string) ([]byte, error) {
	if strings.Contains(id, "..") || strings.ContainsAny(id, "/\\") {
		return nil, fmt.Errorf("invalid version id")
	}
	path := filepath.Join(s.dataDir, versionDir, id)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("version not found")
	}
	return data, nil
}

func (s *FlowVersionService) DiffVersions(fromID, toID string) (*FlowDiff, error) {
	fromData, err := s.GetVersion(fromID)
	if err != nil {
		return nil, fmt.Errorf("from version: %w", err)
	}
	toData, err := s.GetVersion(toID)
	if err != nil {
		return nil, fmt.Errorf("to version: %w", err)
	}

	fromNodes := parseNodeMap(fromData)
	toNodes := parseNodeMap(toData)

	diff := &FlowDiff{}

	for id, node := range toNodes {
		if _, exists := fromNodes[id]; !exists {
			diff.Added = append(diff.Added, nodeEntry(id, node))
		}
	}

	for id, node := range fromNodes {
		if _, exists := toNodes[id]; !exists {
			diff.Removed = append(diff.Removed, nodeEntry(id, node))
		}
	}

	for id, toNode := range toNodes {
		fromNode, exists := fromNodes[id]
		if !exists {
			continue
		}
		changed := diffFields(fromNode, toNode)
		if len(changed) > 0 {
			diff.Modified = append(diff.Modified, FlowDiffMod{
				ID:      id,
				Type:    strField(toNode, "type"),
				Label:   strField(toNode, "label"),
				Changed: changed,
			})
		}
	}

	return diff, nil
}

func (s *FlowVersionService) Revert(id string) error {
	data, err := s.GetVersion(id)
	if err != nil {
		return err
	}

	flowsPath := filepath.Join(s.dataDir, "flows.json")
	if err := os.WriteFile(flowsPath, data, 0600); err != nil {
		return fmt.Errorf("write flows.json: %w", err)
	}

	s.mu.Lock()
	s.lastHash = hashBytes(data)
	s.mu.Unlock()

	return nil
}

func (s *FlowVersionService) pruneOld() {
	dir := filepath.Join(s.dataDir, versionDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}

	if len(files) <= maxVersions {
		return
	}

	sort.Strings(files)
	for _, name := range files[:len(files)-maxVersions] {
		_ = os.Remove(filepath.Join(dir, name))
	}
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func countNodes(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var items []interface{}
	if json.Unmarshal(data, &items) != nil {
		return 0
	}
	return len(items)
}

func parseNodeMap(data []byte) map[string]map[string]interface{} {
	var items []map[string]interface{}
	if json.Unmarshal(data, &items) != nil {
		return nil
	}
	result := make(map[string]map[string]interface{}, len(items))
	for _, item := range items {
		if id, ok := item["id"].(string); ok {
			result[id] = item
		}
	}
	return result
}

func nodeEntry(id string, node map[string]interface{}) FlowDiffEntry {
	return FlowDiffEntry{
		ID:    id,
		Type:  strField(node, "type"),
		Label: strField(node, "label"),
	}
}

func strField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func diffFields(a, b map[string]interface{}) []string {
	var changed []string
	seen := make(map[string]bool)

	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}

	for k := range seen {
		if k == "id" {
			continue
		}
		va, _ := json.Marshal(a[k])
		vb, _ := json.Marshal(b[k])
		if string(va) != string(vb) {
			changed = append(changed, k)
		}
	}

	sort.Strings(changed)
	return changed
}
