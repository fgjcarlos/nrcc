package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// CurrentSchemaVersion is the version written by the versioned JSON stores.
// Only bump it for a change older binaries cannot safely read and rewrite.
const CurrentSchemaVersion = 1

// CheckSchemaVersions validates the versioned files in a live or staged data directory.
func CheckSchemaVersions(dir string) error {
	for _, name := range []string{"config.json", "cc-users.json", "backup_config.json"} {
		if err := CheckSchemaFile(filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	return nil
}

// CheckSchemaFile checks a store file if present, without modifying it.
func CheckSchemaFile(path string) error {
	// #nosec G304 -- callers supply an operator data directory or a private restore staging directory and a fixed filename.
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check schema of %s: %w", filepath.Base(path), err)
	}
	return CheckSchemaVersion(path, data)
}

// CheckSchemaVersion accepts unversioned legacy data. The existing readers
// and recovery paths still own errors unrelated to the version marker.
func CheckSchemaVersion(path string, data []byte) error {
	var header struct {
		Version json.RawMessage `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &header); err != nil || header.Version == nil {
		return nil
	}
	var version *uint64
	if err := json.Unmarshal(header.Version, &version); err != nil || version == nil {
		return fmt.Errorf("%s has an invalid schemaVersion; expected a non-negative integer", filepath.Base(path))
	}
	if *version > CurrentSchemaVersion {
		return fmt.Errorf("%s has schemaVersion %d, but this binary supports up to %d; start with a newer NRCC version", filepath.Base(path), *version, CurrentSchemaVersion)
	}
	return nil
}

// MarshalVersionedJSON adds the on-disk version without changing the API models.
func MarshalVersionedJSON(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, errors.New("versioned JSON must be an object")
	}
	fields["schemaVersion"] = json.RawMessage(fmt.Sprint(CurrentSchemaVersion))
	return json.MarshalIndent(fields, "", "  ")
}
