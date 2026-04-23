package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens the SQLite database, sets pragmas, and runs all migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Set pragmas
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	// Limit to a single connection — SQLite does not benefit from multiple
	// concurrent writers and the extra connections only cause contention.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// migrate applies all pending migrations using PRAGMA user_version
func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}

	migrations := []func(*sql.Tx) error{
		v1InitialSchema,
	}

	for i := version; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", i+1, err)
		}

		if err := migrations[i](tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}

		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("set user_version to %d: %w", i+1, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", i+1, err)
		}
	}

	return nil
}

// v1InitialSchema creates all initial schema: users, audit_logs, sessions, job_history, doctor_runs, config_snapshots
func v1InitialSchema(tx *sql.Tx) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		username TEXT,
		detail TEXT,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS job_history (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		status TEXT NOT NULL,
		started_at TEXT NOT NULL,
		finished_at TEXT,
		triggered_by TEXT,
		summary TEXT,
		error TEXT,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS doctor_runs (
		id TEXT PRIMARY KEY,
		generated_at TEXT NOT NULL,
		overall_status TEXT NOT NULL,
		checks_json TEXT NOT NULL,
		created_at TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS config_snapshots (
		id          TEXT PRIMARY KEY,
		created_at  TEXT NOT NULL,
		label       TEXT NOT NULL DEFAULT '',
		reason      TEXT NOT NULL,
		config_json TEXT NOT NULL DEFAULT '',
		settings_js TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_job_history_type_status ON job_history(type, status);
	CREATE INDEX IF NOT EXISTS idx_job_history_created_at ON job_history(created_at);
	CREATE INDEX IF NOT EXISTS idx_doctor_runs_created_at ON doctor_runs(created_at);
	CREATE INDEX IF NOT EXISTS idx_config_snapshots_created_at ON config_snapshots(created_at DESC);
	`

	if _, err := tx.Exec(schema); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}

	return nil
}
