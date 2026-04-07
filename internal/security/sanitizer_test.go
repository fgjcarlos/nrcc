package security

import (
	"testing"
)

func TestSanitizeMapRedactsSensitiveKeys(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	testCases := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "password redaction",
			input: map[string]interface{}{
				"password": "secret123",
				"username": "alice",
			},
			expected: map[string]interface{}{
				"password": "[REDACTED]",
				"username": "alice",
			},
		},
		{
			name: "token redaction",
			input: map[string]interface{}{
				"token":  "eyJhbGc...",
				"status": "active",
			},
			expected: map[string]interface{}{
				"token":  "[REDACTED]",
				"status": "active",
			},
		},
		{
			name: "secret redaction",
			input: map[string]interface{}{
				"secret": "my-secret-key",
				"name":   "config",
			},
			expected: map[string]interface{}{
				"secret": "[REDACTED]",
				"name":   "config",
			},
		},
		{
			name: "multiple sensitive keys",
			input: map[string]interface{}{
				"password":   "pass123",
				"token":      "tok123",
				"secret":     "sec123",
				"key":        "mykey",
				"cookie":     "cook123",
				"session":    "sess123",
				"auth":       "auth123",
				"credential": "cred123",
				"pass":       "pass456",
				"username":   "alice",
			},
			expected: map[string]interface{}{
				"password":   "[REDACTED]",
				"token":      "[REDACTED]",
				"secret":     "[REDACTED]",
				"key":        "[REDACTED]",
				"cookie":     "[REDACTED]",
				"session":    "[REDACTED]",
				"auth":       "[REDACTED]",
				"credential": "[REDACTED]",
				"pass":       "[REDACTED]",
				"username":   "alice",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizer.SanitizeMap(tc.input)
			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d keys, got %d", len(tc.expected), len(result))
			}

			for key, expectedVal := range tc.expected {
				if result[key] != expectedVal {
					t.Errorf("key %s: expected %v, got %v", key, expectedVal, result[key])
				}
			}
		})
	}
}

func TestSanitizeMapNilMap(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	result := sanitizer.SanitizeMap(nil)
	if result != nil {
		t.Fatal("expected nil for nil map")
	}
}

func TestSanitizeMapRecursive(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	input := map[string]interface{}{
		"database": map[string]interface{}{
			"password": "secret123",
			"username": "admin",
			"host":     "localhost",
		},
		"api": map[string]interface{}{
			"token":  "mytoken",
			"secret": "mysecret",
		},
		"user": "alice",
	}

	result := sanitizer.SanitizeMap(input)

	// Check top-level non-sensitive key
	if result["user"] != "alice" {
		t.Errorf("expected 'alice', got %v", result["user"])
	}

	// Check nested map
	dbMap, ok := result["database"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested map for 'database'")
	}
	if dbMap["password"] != "[REDACTED]" {
		t.Errorf("expected '[REDACTED]' for nested password, got %v", dbMap["password"])
	}
	if dbMap["username"] != "admin" {
		t.Errorf("expected 'admin' for username, got %v", dbMap["username"])
	}

	// Check nested api map
	apiMap, ok := result["api"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested map for 'api'")
	}
	if apiMap["token"] != "[REDACTED]" {
		t.Errorf("expected '[REDACTED]' for nested token, got %v", apiMap["token"])
	}
	if apiMap["secret"] != "[REDACTED]" {
		t.Errorf("expected '[REDACTED]' for nested secret, got %v", apiMap["secret"])
	}
}

func TestSanitizeMapCaseInsensitive(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	input := map[string]interface{}{
		"Password": "secret123",
		"TOKEN":    "mytoken",
		"Secret":   "mysecret",
		"KEY":      "mykey",
	}

	result := sanitizer.SanitizeMap(input)

	// Case-insensitive redaction should work
	if result["Password"] != "[REDACTED]" {
		t.Errorf("expected '[REDACTED]' for Password, got %v", result["Password"])
	}
	if result["TOKEN"] != "[REDACTED]" {
		t.Errorf("expected '[REDACTED]' for TOKEN, got %v", result["TOKEN"])
	}
	if result["Secret"] != "[REDACTED]" {
		t.Errorf("expected '[REDACTED]' for Secret, got %v", result["Secret"])
	}
	if result["KEY"] != "[REDACTED]" {
		t.Errorf("expected '[REDACTED]' for KEY, got %v", result["KEY"])
	}
}

func TestSanitizeStringSafeStrings(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "plain text",
			input: "This is a safe string",
		},
		{
			name:  "json without secrets",
			input: `{"name":"alice","status":"active"}`,
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "url without credentials",
			input: "https://example.com/api/v1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizer.SanitizeString(tc.input)
			if result != tc.input {
				t.Errorf("expected unchanged string, got %s", result)
			}
		})
	}
}

func TestSanitizeStringJWT(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	// Valid JWT pattern: eyJ...eyJ...
	jwtToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	input := "Authorization header: " + jwtToken
	result := sanitizer.SanitizeString(input)

	if result == input {
		t.Errorf("expected JWT to be redacted, but got original string")
	}
	if !contains(result, "[REDACTED-JWT]") {
		t.Errorf("expected '[REDACTED-JWT]' in result, got %s", result)
	}
}

func TestSanitizeStringBearerToken(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	result := sanitizer.SanitizeString(input)

	if result == input {
		t.Errorf("expected Bearer token to be redacted")
	}
	if !contains(result, "Bearer [REDACTED]") {
		t.Errorf("expected 'Bearer [REDACTED]' in result, got %s", result)
	}
}

func TestSanitizeStringUrlCredentials(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	input := "postgresql://admin:secretpassword@localhost:5432/mydb"
	result := sanitizer.SanitizeString(input)

	if result == input {
		t.Errorf("expected URL credentials to be redacted")
	}
	if !contains(result, "://[REDACTED]:[REDACTED]@") {
		t.Errorf("expected '://[REDACTED]:[REDACTED]@' in result, got %s", result)
	}
}

func TestSanitizeStringAPIKey(t *testing.T) {
	t.Parallel()

	sanitizer := NewSanitizer()

	input := `{"api_key":"sk-12345abcde","token":"mytoken123","secret":"mysecret"}`
	result := sanitizer.SanitizeString(input)

	if result == input {
		t.Errorf("expected API key patterns to be redacted")
	}
	if contains(result, "sk-12345abcde") {
		t.Errorf("expected api_key value to be redacted, but it's still in %s", result)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
