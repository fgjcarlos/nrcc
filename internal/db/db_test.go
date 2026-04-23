package db

import (
	"database/sql"
	"os"
	"testing"
)

func TestOpen_CreatesDatabase(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	db, err := Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Verify database is working
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestOpen_SetsPragmas(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	db, err := Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Check WAL mode
	var journalMode string
	db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if journalMode != "wal" {
		t.Errorf("Expected WAL mode, got %s", journalMode)
	}

	// Check foreign_keys
	var fk int
	db.QueryRow("PRAGMA foreign_keys").Scan(&fk)
	if fk != 1 {
		t.Errorf("Expected foreign_keys=1, got %d", fk)
	}
}

func TestOpen_RunsMigrations(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	db, err := Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Verify all tables exist
	tables := []string{
		"users",
		"audit_logs",
		"sessions",
		"job_history",
		"doctor_runs",
		"config_snapshots",
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count == 0 {
			t.Errorf("Table %s not created", table)
		}
	}
}

func TestOpen_SetsUserVersion(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	db, err := Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Check user_version (should be 1 after first migration)
	var version int
	db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 1 {
		t.Errorf("Expected user_version=1, got %d", version)
	}
}

func TestOpen_IdempotentMigrations(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Open database first time
	db1, err := Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("First Open failed: %v", err)
	}
	db1.Close()

	// Open database second time - should not fail or re-run migrations
	db2, err := Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Second Open failed: %v", err)
	}
	defer db2.Close()

	// Verify user_version is still 1 (no re-run)
	var version int
	db2.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 1 {
		t.Errorf("Expected user_version=1, got %d", version)
	}
}

func TestV1InitialSchema_CreatesTables(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create a transaction and run migration manually
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	if err := v1InitialSchema(tx); err != nil {
		t.Fatalf("v1InitialSchema failed: %v", err)
	}

	// Verify tables exist
	tables := []string{
		"users",
		"audit_logs",
		"sessions",
		"job_history",
		"doctor_runs",
		"config_snapshots",
	}

	for _, table := range tables {
		var name string
		err := tx.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s not created: %v", table, err)
		}
	}
}

func TestV1InitialSchema_CreatesBothWithoutErrors(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Enable foreign_keys
	db.Exec("PRAGMA foreign_keys = ON")

	// Run migration twice to test IF NOT EXISTS
	tx1, _ := db.Begin()
	v1InitialSchema(tx1)
	tx1.Commit()

	tx2, _ := db.Begin()
	err = v1InitialSchema(tx2)
	tx2.Commit()

	if err != nil {
		t.Errorf("Second migration failed: %v", err)
	}
}
