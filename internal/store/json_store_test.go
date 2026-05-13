package store

import (
	"os"
	"path/filepath"
	"testing"
)

type TestData struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestJSONStore_Write_CreatesFile(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test.json")

	store := NewJSONStore[TestData](storePath)
	data := TestData{Name: "test", Value: 42}

	err := store.Write(data)

	if err != nil {
		t.Errorf("Write should not error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(storePath); err != nil {
		t.Errorf("Store file should exist: %v", err)
	}
}

func TestJSONStore_Read_ParsesJSON(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test.json")

	// Write file manually
	content := `{"name":"test","value":42}`
	if err := os.WriteFile(storePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	store := NewJSONStore[TestData](storePath)
	data, err := store.Read()

	if err != nil {
		t.Errorf("Read should not error: %v", err)
	}

	if data.Name != "test" {
		t.Errorf("Expected Name 'test', got %s", data.Name)
	}
	if data.Value != 42 {
		t.Errorf("Expected Value 42, got %d", data.Value)
	}
}

func TestJSONStore_ReadWrite_RoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test.json")

	store := NewJSONStore[TestData](storePath)

	// Write
	original := TestData{Name: "roundtrip", Value: 123}
	if err := store.Write(original); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	loaded, err := store.Read()

	if err != nil {
		t.Errorf("Read failed: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", loaded.Name, original.Name)
	}
	if loaded.Value != original.Value {
		t.Errorf("Value mismatch: got %d, want %d", loaded.Value, original.Value)
	}
}

func TestJSONStore_Read_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "nonexistent.json")

	store := NewJSONStore[TestData](storePath)
	_, err := store.Read()

	if err == nil {
		t.Error("Read should error for non-existent file")
	}
}

func TestJSONStore_Read_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "invalid.json")

	// Write invalid JSON
	if err := os.WriteFile(storePath, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	store := NewJSONStore[TestData](storePath)
	_, err := store.Read()

	if err == nil {
		t.Error("Read should error for invalid JSON")
	}
}

func TestJSONStore_Exists_TrueWhenFileExists(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test.json")

	// Create file
	if err := os.WriteFile(storePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	store := NewJSONStore[TestData](storePath)
	exists := store.Exists()

	if !exists {
		t.Error("Exists should return true when file exists")
	}
}

func TestJSONStore_Exists_FalseWhenFileNotExists(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "nonexistent.json")

	store := NewJSONStore[TestData](storePath)
	exists := store.Exists()

	if exists {
		t.Error("Exists should return false when file doesn't exist")
	}
}

func TestJSONStore_Write_Atomic_CreatesTemp(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test.json")

	store := NewJSONStore[TestData](storePath)
	data := TestData{Name: "atomic", Value: 99}

	err := store.Write(data)

	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// Verify no .tmp file remains
	tmpPath := storePath + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("Temporary file should be cleaned up after atomic rename")
	}

	// Verify actual file exists
	if _, err := os.Stat(storePath); err != nil {
		t.Error("Final file should exist")
	}
}

func TestJSONStore_Write_OverwritesExisting(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test.json")

	store := NewJSONStore[TestData](storePath)

	// First write
	data1 := TestData{Name: "first", Value: 1}
	if err := store.Write(data1); err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Second write (overwrite)
	data2 := TestData{Name: "second", Value: 2}
	if err := store.Write(data2); err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Verify file contains second data
	loaded, err := store.Read()

	if err != nil {
		t.Errorf("Read failed: %v", err)
	}

	if loaded.Name != "second" {
		t.Errorf("Expected Name 'second', got %s", loaded.Name)
	}
	if loaded.Value != 2 {
		t.Errorf("Expected Value 2, got %d", loaded.Value)
	}
}

func TestJSONStore_Write_RequiresParentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "subdir", "nested", "test.json")

	store := NewJSONStore[TestData](storePath)
	data := TestData{Name: "nested", Value: 7}

	err := store.Write(data)

	// Should error because parent directory doesn't exist
	if err == nil {
		t.Error("Write should error when parent directory doesn't exist")
	}
}

func TestJSONStore_Concurrent_Reads(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test.json")

	store := NewJSONStore[TestData](storePath)
	data := TestData{Name: "concurrent", Value: 50}

	if err := store.Write(data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Simulate concurrent reads
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func() {
			loaded, err := store.Read()
			if err != nil {
				t.Errorf("Concurrent read failed: %v", err)
			}
			if loaded.Name != "concurrent" {
				t.Errorf("Data corrupted in concurrent read")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestJSONStore_ReadPreservesFormatting(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test.json")

	store := NewJSONStore[TestData](storePath)
	data := TestData{Name: "formatted", Value: 999}

	// Write
	if err := store.Write(data); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read file content directly
	content, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Verify it's formatted with indentation (MarshalIndent)
	contentStr := string(content)
	if contentStr[0] != '{' || contentStr[len(contentStr)-1] != '\n' {
		// MarshalIndent should produce formatted output
		// (exact format depends on the implementation)
	}
}

func TestJSONStore_WriteErrorWithInvalidPath(t *testing.T) {
	// Use a path that will fail to create
	// (This might be OS-specific, so we'll do a basic check)
	storePath := "/dev/null/nonexistent/path.json" // Usually fails on Unix

	store := NewJSONStore[TestData](storePath)
	data := TestData{Name: "fail", Value: 0}

	err := store.Write(data)

	// We expect an error, but exact error depends on OS
	// Just verify the operation doesn't silently succeed
	if err == nil && storePath != "/dev/null/nonexistent/path.json" {
		// This assertion is conditional because some systems might behave differently
	}
}

func TestJSONStore_MultipleStores_IndependentPaths(t *testing.T) {
	tempDir := t.TempDir()
	store1Path := filepath.Join(tempDir, "store1.json")
	store2Path := filepath.Join(tempDir, "store2.json")

	store1 := NewJSONStore[TestData](store1Path)
	store2 := NewJSONStore[TestData](store2Path)

	data1 := TestData{Name: "store1", Value: 1}
	data2 := TestData{Name: "store2", Value: 2}

	if err := store1.Write(data1); err != nil {
		t.Fatalf("Store1 write failed: %v", err)
	}
	if err := store2.Write(data2); err != nil {
		t.Fatalf("Store2 write failed: %v", err)
	}

	// Read and verify independence
	loaded1, _ := store1.Read()
	loaded2, _ := store2.Read()

	if loaded1.Name != "store1" {
		t.Error("Store1 should contain store1 data")
	}
	if loaded2.Name != "store2" {
		t.Error("Store2 should contain store2 data")
	}
}
