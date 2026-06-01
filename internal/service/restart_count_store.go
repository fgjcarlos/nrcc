package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/composedof2/nrcc/internal/ui"
)

// restartCountStore persists the cumulative Node-RED auto-restart counter to
// a JSON file under the data directory. Writes are atomic: data is written to
// a temporary file and then renamed into place so a crash mid-write never
// leaves a torn file.
type restartCountStore struct {
	path string
	mu   sync.Mutex
}

// restartCountPayload is the JSON schema for the persisted file.
// Using a struct (not a bare int) lets us add fields later without migration.
type restartCountPayload struct {
	CumulativeRestarts int `json:"cumulativeRestarts"`
}

// newRestartCountStore constructs a store whose backing file is
// <dataDir>/restart_count.json.
func newRestartCountStore(dataDir string) *restartCountStore {
	return &restartCountStore{
		path: filepath.Join(dataDir, "restart_count.json"),
	}
}

// Load reads the persisted cumulative restart count. On any error (missing
// file, corrupt JSON, permission denied) it returns 0 and logs a warning —
// it NEVER returns an error to the caller so startup is never blocked.
func (s *restartCountStore) Load() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			ui.Warnf("restart count store: read %s: %v — treating as 0", s.path, err)
		}
		return 0
	}

	var payload restartCountPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		ui.Warnf("restart count store: parse %s: %v — treating as 0", s.path, err)
		return 0
	}

	return payload.CumulativeRestarts
}

// Save persists n as the cumulative restart count. The write is atomic: data
// goes to a .tmp file first, then os.Rename moves it into place.
func (s *restartCountStore) Save(n int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(restartCountPayload{CumulativeRestarts: n})
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}
