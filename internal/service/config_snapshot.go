package service

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"nrcc/internal/model"
)

// SnapshotService handles config_snapshots database operations
type SnapshotService struct {
	db *sql.DB
}

// NewSnapshotService creates a new snapshot service
func NewSnapshotService(db *sql.DB) *SnapshotService {
	return &SnapshotService{db: db}
}

// InitConfigSnapshotSchema creates the config_snapshots table if it doesn't exist
func InitConfigSnapshotSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS config_snapshots (
		id          TEXT PRIMARY KEY,
		created_at  TEXT NOT NULL,
		label       TEXT NOT NULL DEFAULT '',
		reason      TEXT NOT NULL,
		config_json TEXT NOT NULL DEFAULT '',
		settings_js TEXT NOT NULL DEFAULT ''
	);
	CREATE INDEX IF NOT EXISTS idx_config_snapshots_created_at ON config_snapshots(created_at DESC);
	`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("initialize config snapshot schema: %w", err)
	}
	return nil
}

// CreateSnapshot creates a new config snapshot and enforces retention limit
func (s *SnapshotService) CreateSnapshot(label, reason, configJSON, settingsJS string) (model.ConfigSnapshot, error) {
	id := uuid.New().String()
	createdAt := time.Now().Format(time.RFC3339)

	// Insert snapshot
	if _, err := s.db.Exec(
		`INSERT INTO config_snapshots (id, created_at, label, reason, config_json, settings_js)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, createdAt, label, reason, configJSON, settingsJS,
	); err != nil {
		return model.ConfigSnapshot{}, fmt.Errorf("insert snapshot: %w", err)
	}

	// Enforce retention limit: keep only 50 most recent snapshots
	const maxSnapshots = 50
	if _, err := s.db.Exec(`
		DELETE FROM config_snapshots WHERE id IN (
			SELECT id FROM config_snapshots
			ORDER BY created_at DESC
			LIMIT -1 OFFSET ?
		)`, maxSnapshots); err != nil {
		return model.ConfigSnapshot{}, fmt.Errorf("enforce snapshot retention: %w", err)
	}

	return model.ConfigSnapshot{
		ID:        id,
		CreatedAt: createdAt,
		Label:     label,
		Reason:    reason,
	}, nil
}

// ListSnapshots returns up to 50 most recent config snapshots (without content)
func (s *SnapshotService) ListSnapshots() (model.ConfigSnapshotList, error) {
	rows, err := s.db.Query(`
		SELECT id, created_at, label, reason
		FROM config_snapshots
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		return model.ConfigSnapshotList{}, fmt.Errorf("query snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []model.ConfigSnapshot
	for rows.Next() {
		var snapshot model.ConfigSnapshot
		if err := rows.Scan(&snapshot.ID, &snapshot.CreatedAt, &snapshot.Label, &snapshot.Reason); err != nil {
			return model.ConfigSnapshotList{}, fmt.Errorf("scan snapshot row: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return model.ConfigSnapshotList{}, fmt.Errorf("iterate snapshot rows: %w", err)
	}

	return model.ConfigSnapshotList{Items: snapshots}, nil
}

// GetSnapshot retrieves a full snapshot by ID (including content)
func (s *SnapshotService) GetSnapshot(id string) (*model.ConfigSnapshot, error) {
	var snapshot model.ConfigSnapshot
	err := s.db.QueryRow(`
		SELECT id, created_at, label, reason, config_json, settings_js
		FROM config_snapshots
		WHERE id = ?
	`, id).Scan(&snapshot.ID, &snapshot.CreatedAt, &snapshot.Label, &snapshot.Reason, &snapshot.ConfigJSON, &snapshot.SettingsJS)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query snapshot: %w", err)
	}

	return &snapshot, nil
}

// DeleteSnapshot deletes a snapshot by ID
func (s *SnapshotService) DeleteSnapshot(id string) error {
	if _, err := s.db.Exec(`DELETE FROM config_snapshots WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete snapshot: %w", err)
	}
	return nil
}
