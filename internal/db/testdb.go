package db

import "database/sql"

// OpenMemory opens an in-memory SQLite database with all migrations applied.
// For use in tests only.
func OpenMemory() (*sql.DB, error) {
	return Open(":memory:")
}
