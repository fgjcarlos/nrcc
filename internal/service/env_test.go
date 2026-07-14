package service

import (
	"os"
	"testing"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// TestValidateValue tests the ValidateValue function with table-driven tests
func TestValidateValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		typ     string
		wantErr bool
	}{
		// String type — accepts any value
		{"string: empty", "", "string", false},
		{"string: simple", "hello", "string", false},
		{"string: with spaces", "hello world", "string", false},
		{"string: with special chars", "!@#$%^&*()", "string", false},

		// Secret type — accepts any value
		{"secret: empty", "", "secret", false},
		{"secret: simple", "password123", "secret", false},
		{"secret: with special chars", "p@$$w0rd!", "secret", false},

		// Number type — must be valid numeric
		{"number: valid integer", "42", "number", false},
		{"number: valid float", "3.14", "number", false},
		{"number: negative", "-100", "number", false},
		{"number: scientific", "1e10", "number", false},
		{"number: empty", "", "number", true},
		{"number: invalid text", "not-a-number", "number", true},
		{"number: with letters", "123abc", "number", true},

		// Boolean type — must be "true" or "false"
		{"boolean: true", "true", "boolean", false},
		{"boolean: false", "false", "boolean", false},
		{"boolean: empty", "", "boolean", true},
		{"boolean: invalid", "yes", "boolean", true},
		{"boolean: invalid 1", "1", "boolean", true},
		{"boolean: invalid 0", "0", "boolean", true},

		// Unknown type
		{"unknown type", "value", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValue(tt.value, tt.typ)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNormalizeValue tests the NormalizeValue function with table-driven tests
func TestNormalizeValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		typ     string
		want    string
		wantErr bool
	}{
		// String type — returns as-is
		{"string: empty", "", "string", "", false},
		{"string: simple", "hello", "string", "hello", false},
		{"string: with spaces", "  hello  ", "string", "  hello  ", false},

		// Secret type — returns as-is
		{"secret: empty", "", "secret", "", false},
		{"secret: password", "secret123", "secret", "secret123", false},

		// Number type — validates and returns as-is
		{"number: integer", "42", "number", "42", false},
		{"number: float", "3.14", "number", "3.14", false},
		{"number: negative", "-100", "number", "-100", false},
		{"number: scientific", "1e10", "number", "1e10", false},
		{"number: empty", "", "number", "", true},
		{"number: invalid", "not-a-number", "number", "", true},

		// Boolean type — normalizes to "true" or "false"
		{"boolean: true", "true", "boolean", "true", false},
		{"boolean: false", "false", "boolean", "false", false},
		{"boolean: TRUE uppercase", "TRUE", "boolean", "true", false},
		{"boolean: False mixed", "False", "boolean", "false", false},
		{"boolean: with spaces", "  true  ", "boolean", "true", false},
		{"boolean: empty", "", "boolean", "", true},
		{"boolean: yes", "yes", "boolean", "", true},
		{"boolean: 1", "1", "boolean", "", true},

		// Unknown type
		{"unknown type", "value", "unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeValue(tt.value, tt.typ)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMigrateEnvTypes tests the MigrateEnvTypes function with table-driven tests
func TestMigrateEnvTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    model.EnvVar
		expected model.EnvVar
	}{
		// Empty type (legacy) → migrates to "string"
		{
			name: "legacy empty type",
			input: model.EnvVar{
				Key:   "MY_VAR",
				Value: "value",
				Type:  "",
			},
			expected: model.EnvVar{
				Key:   "MY_VAR",
				Value: "value",
				Type:  "string",
			},
		},

		// "plain" type (legacy) → migrates to "string"
		{
			name: "legacy plain type",
			input: model.EnvVar{
				Key:   "MY_VAR",
				Value: "value",
				Type:  "plain",
			},
			expected: model.EnvVar{
				Key:   "MY_VAR",
				Value: "value",
				Type:  "string",
			},
		},

		// Encrypted with empty type → migrates to "secret"
		{
			name: "encrypted empty type",
			input: model.EnvVar{
				Key:       "PASSWORD",
				Value:     "encrypted_value",
				Type:      "",
				Encrypted: true,
			},
			expected: model.EnvVar{
				Key:       "PASSWORD",
				Value:     "encrypted_value",
				Type:      "secret",
				Encrypted: true,
			},
		},

		// Encrypted with "plain" type → migrates to "secret"
		{
			name: "encrypted plain type",
			input: model.EnvVar{
				Key:       "PASSWORD",
				Value:     "encrypted_value",
				Type:      "plain",
				Encrypted: true,
			},
			expected: model.EnvVar{
				Key:       "PASSWORD",
				Value:     "encrypted_value",
				Type:      "secret",
				Encrypted: true,
			},
		},

		// Already modern type — no change
		{
			name: "modern string type",
			input: model.EnvVar{
				Key:   "MY_VAR",
				Value: "value",
				Type:  "string",
			},
			expected: model.EnvVar{
				Key:   "MY_VAR",
				Value: "value",
				Type:  "string",
			},
		},

		// Number type — no change
		{
			name: "number type",
			input: model.EnvVar{
				Key:   "COUNT",
				Value: "42",
				Type:  "number",
			},
			expected: model.EnvVar{
				Key:   "COUNT",
				Value: "42",
				Type:  "number",
			},
		},

		// Boolean type — no change
		{
			name: "boolean type",
			input: model.EnvVar{
				Key:   "ENABLED",
				Value: "true",
				Type:  "boolean",
			},
			expected: model.EnvVar{
				Key:   "ENABLED",
				Value: "true",
				Type:  "boolean",
			},
		},

		// Secret type with encrypted flag — no change
		{
			name: "secret type encrypted",
			input: model.EnvVar{
				Key:       "API_KEY",
				Value:     "secret_key",
				Type:      "secret",
				Encrypted: true,
			},
			expected: model.EnvVar{
				Key:       "API_KEY",
				Value:     "secret_key",
				Type:      "secret",
				Encrypted: true,
			},
		},

		// Encrypted with secret type — no change
		{
			name: "encrypted with secret type",
			input: model.EnvVar{
				Key:       "SECRET",
				Value:     "value",
				Type:      "secret",
				Encrypted: true,
			},
			expected: model.EnvVar{
				Key:       "SECRET",
				Value:     "value",
				Type:      "secret",
				Encrypted: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MigrateEnvTypes(tt.input)
			if got.Key != tt.expected.Key {
				t.Errorf("Key: got %v, want %v", got.Key, tt.expected.Key)
			}
			if got.Value != tt.expected.Value {
				t.Errorf("Value: got %v, want %v", got.Value, tt.expected.Value)
			}
			if got.Type != tt.expected.Type {
				t.Errorf("Type: got %v, want %v", got.Type, tt.expected.Type)
			}
			if got.Encrypted != tt.expected.Encrypted {
				t.Errorf("Encrypted: got %v, want %v", got.Encrypted, tt.expected.Encrypted)
			}
		})
	}
}

func TestParseEnvFile_EmptyFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()
	_ = tmpfile.Close()

	result, err := parseEnvFile(tmpfile.Name())
	if err != nil {
		t.Errorf("parseEnvFile failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

// TestParseEnvFile_SimpleKeyValues tests parsing simple KEY=VALUE pairs
func TestParseEnvFile_SimpleKeyValues(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `KEY1=value1
KEY2=value2
KEY3=value3`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	result, err := parseEnvFile(tmpfile.Name())
	if err != nil {
		t.Errorf("parseEnvFile failed: %v", err)
	}

	expected := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}

	for k, v := range expected {
		if result[k] != v {
			t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
		}
	}
}

// TestParseEnvFile_WithComments tests that comments are ignored
func TestParseEnvFile_WithComments(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `# This is a comment
KEY1=value1
# Another comment
KEY2=value2`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	result, err := parseEnvFile(tmpfile.Name())
	if err != nil {
		t.Errorf("parseEnvFile failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if result["KEY1"] != "value1" {
		t.Errorf("expected KEY1=value1, got KEY1=%s", result["KEY1"])
	}
}

// TestParseEnvFile_WithQuotes tests that quoted values preserve spaces
func TestParseEnvFile_WithQuotes(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `KEY1="value with spaces"
KEY2=simple
KEY3="another quoted value"`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	result, err := parseEnvFile(tmpfile.Name())
	if err != nil {
		t.Errorf("parseEnvFile failed: %v", err)
	}

	if result["KEY1"] != "value with spaces" {
		t.Errorf("expected 'value with spaces', got '%s'", result["KEY1"])
	}
	if result["KEY2"] != "simple" {
		t.Errorf("expected 'simple', got '%s'", result["KEY2"])
	}
	if result["KEY3"] != "another quoted value" {
		t.Errorf("expected 'another quoted value', got '%s'", result["KEY3"])
	}
}

// TestParseEnvFile_ValueWithEquals tests that values can contain = signs
func TestParseEnvFile_ValueWithEquals(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `KEY1=value=with=equals
CONNECTION_STRING=user=admin;pass=secret123`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	result, err := parseEnvFile(tmpfile.Name())
	if err != nil {
		t.Errorf("parseEnvFile failed: %v", err)
	}

	if result["KEY1"] != "value=with=equals" {
		t.Errorf("expected 'value=with=equals', got '%s'", result["KEY1"])
	}
	if result["CONNECTION_STRING"] != "user=admin;pass=secret123" {
		t.Errorf("expected 'user=admin;pass=secret123', got '%s'", result["CONNECTION_STRING"])
	}
}

// TestParseEnvFile_EmptyLines tests that empty lines are ignored
func TestParseEnvFile_EmptyLines(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `KEY1=value1

KEY2=value2

KEY3=value3`
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	result, err := parseEnvFile(tmpfile.Name())
	if err != nil {
		t.Errorf("parseEnvFile failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 entries, got %d", len(result))
	}
}

// TestParseEnvFile_NonExistentFile tests that a non-existent file returns empty map
func TestParseEnvFile_NonExistentFile(t *testing.T) {
	result, err := parseEnvFile("/nonexistent/path/.env")
	if err != nil {
		t.Errorf("parseEnvFile should not error for non-existent file, got: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map for non-existent file, got %d entries", len(result))
	}
}

// TestParseEnvFile_WhitespaceHandling tests whitespace trimming
func TestParseEnvFile_WhitespaceHandling(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	content := `  KEY1  =  value1  
KEY2=value2
   KEY3=value3   `
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	result, err := parseEnvFile(tmpfile.Name())
	if err != nil {
		t.Errorf("parseEnvFile failed: %v", err)
	}

	// Keys are trimmed, values preserve trailing content but line is trimmed first
	if result["KEY1"] != "  value1" {
		t.Errorf("expected '  value1', got '%s'", result["KEY1"])
	}
	if result["KEY2"] != "value2" {
		t.Errorf("expected 'value2', got '%s'", result["KEY2"])
	}
	if result["KEY3"] != "value3" {
		t.Errorf("expected 'value3', got '%s'", result["KEY3"])
	}
}
