package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/composedof2/nrcc/internal/model"
	"github.com/composedof2/nrcc/internal/service"
)

// TestPostEnvValidationFlow tests the PostEnv handler's validation and normalization
// This covers Domain 6 validation scenarios from the spec
func TestPostEnvValidationFlow(t *testing.T) {
	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		expectedError  string
		expectedType   string
		expectedValue  string
	}{
		// String type tests
		{
			name: "Valid string value",
			payload: map[string]interface{}{
				"key":   "STRING_VAR",
				"value": "hello world",
				"type":  "string",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "string",
			expectedValue:  "hello world",
		},
		{
			name: "Empty string is valid",
			payload: map[string]interface{}{
				"key":   "EMPTY_STRING",
				"value": "",
				"type":  "string",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "string",
			expectedValue:  "",
		},

		// Number type tests
		{
			name: "Valid integer number",
			payload: map[string]interface{}{
				"key":   "PORT_NUMBER",
				"value": "8080",
				"type":  "number",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "number",
			expectedValue:  "8080",
		},
		{
			name: "Valid float number",
			payload: map[string]interface{}{
				"key":   "FLOAT_VAR",
				"value": "3.14159",
				"type":  "number",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "number",
			expectedValue:  "3.14159",
		},
		{
			name: "Invalid number — non-numeric string",
			payload: map[string]interface{}{
				"key":   "BAD_NUMBER",
				"value": "not-a-number",
				"type":  "number",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_VALUE",
		},
		{
			name: "Invalid number — empty value",
			payload: map[string]interface{}{
				"key":   "EMPTY_NUMBER",
				"value": "",
				"type":  "number",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_VALUE",
		},

		// Boolean type tests
		{
			name: "Valid boolean true",
			payload: map[string]interface{}{
				"key":   "BOOL_TRUE",
				"value": "true",
				"type":  "boolean",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "boolean",
			expectedValue:  "true",
		},
		{
			name: "Valid boolean false",
			payload: map[string]interface{}{
				"key":   "BOOL_FALSE",
				"value": "false",
				"type":  "boolean",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "boolean",
			expectedValue:  "false",
		},
		{
			name: "Invalid boolean — yes instead of true",
			payload: map[string]interface{}{
				"key":   "BAD_BOOL",
				"value": "yes",
				"type":  "boolean",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_VALUE",
		},
		{
			name: "Invalid boolean — numeric 1",
			payload: map[string]interface{}{
				"key":   "BAD_BOOL_1",
				"value": "1",
				"type":  "boolean",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_VALUE",
		},

		// Secret type tests
		{
			name: "Valid secret value",
			payload: map[string]interface{}{
				"key":   "API_SECRET",
				"value": "super-secret-key",
				"type":  "secret",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "secret",
			expectedValue:  "super-secret-key",
		},
		{
			name: "Secret with special characters",
			payload: map[string]interface{}{
				"key":   "DB_PASSWORD",
				"value": "p@$$w0rd!#&",
				"type":  "secret",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "secret",
			expectedValue:  "p@$$w0rd!#&",
		},

		// Default type (missing type field)
		{
			name: "Missing type field defaults to string",
			payload: map[string]interface{}{
				"key":   "LEGACY_VAR",
				"value": "legacy-value",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "string",
			expectedValue:  "legacy-value",
		},

		// Normalization tests
		{
			name: "Boolean normalization — uppercase TRUE to lowercase true",
			payload: map[string]interface{}{
				"key":   "BOOL_UPPER",
				"value": "TRUE",
				"type":  "boolean",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "boolean",
			expectedValue:  "true", // should be normalized
		},
		{
			name: "Boolean normalization — mixed case False to false",
			payload: map[string]interface{}{
				"key":   "BOOL_MIXED",
				"value": "False",
				"type":  "boolean",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "boolean",
			expectedValue:  "false",
		},
		{
			name: "Boolean normalization — with spaces",
			payload: map[string]interface{}{
				"key":   "BOOL_SPACES",
				"value": "  true  ",
				"type":  "boolean",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "boolean",
			expectedValue:  "true",
		},

		// Description field
		{
			name: "Variable with description",
			payload: map[string]interface{}{
				"key":         "DOCUMENTED_VAR",
				"value":       "some-value",
				"type":        "string",
				"description": "This is a documented variable",
			},
			expectedStatus: http.StatusOK,
			expectedType:   "string",
			expectedValue:  "some-value",
		},

		// Error cases
		{
			name: "Missing key field",
			payload: map[string]interface{}{
				"value": "something",
				"type":  "string",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			configSvc := service.NewConfigService(t.TempDir())
			envSvc := service.NewEnvService(configSvc)
			handler := NewEnvHandler(envSvc, t.TempDir())

			// Create request
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/env", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.PostEnv(w, req)

			// Check HTTP status
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d\nResponse: %s",
					tt.expectedStatus, w.Code, w.Body.String())
				return
			}

			// If error is expected, verify error code
			if tt.expectedStatus != http.StatusOK {
				var errResp struct {
					Success bool `json:"success"`
					Error   *struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					} `json:"error"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}
				if errResp.Error == nil {
					t.Errorf("Expected error response, got: %s", w.Body.String())
					return
				}
				if errResp.Error.Code != tt.expectedError {
					t.Errorf("Expected error code %s, got %s", tt.expectedError, errResp.Error.Code)
				}
				return
			}

			// Verify the value was stored correctly
			savedVars, err := envSvc.List()
			if err != nil {
				t.Fatalf("Failed to list env vars: %v", err)
			}

			// Find the saved variable
			key := tt.payload["key"].(string)
			var found *model.EnvVar
			for i := range savedVars {
				if savedVars[i].Key == key {
					found = &savedVars[i]
					break
				}
			}

			if found == nil {
				t.Errorf("Variable %s not found in saved config", key)
				return
			}

			// Check type
			if found.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, found.Type)
			}

			// Check value (Note: secret values are masked in List() response)
			if tt.expectedType != "secret" && found.Value != tt.expectedValue {
				t.Errorf("Expected value %q, got %q", tt.expectedValue, found.Value)
			}
		})
	}
}

// TestPostEnvHandlerRoundTrip tests a complete cycle: POST and then GET
func TestPostEnvHandlerRoundTrip(t *testing.T) {
	configSvc := service.NewConfigService(t.TempDir())
	envSvc := service.NewEnvService(configSvc)
	handler := NewEnvHandler(envSvc, t.TempDir())

	// POST a new environment variable
	payload := map[string]interface{}{
		"key":         "DATABASE_HOST",
		"value":       "localhost",
		"type":        "string",
		"description": "Database hostname",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/env", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.PostEnv(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to POST env var: %d — %s", w.Code, w.Body.String())
	}

	// GET the environment variables
	req = httptest.NewRequest("GET", "/api/env", nil)
	w = httptest.NewRecorder()
	handler.GetEnv(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to GET env vars: %d", w.Code)
	}

	var resp struct {
		Data []model.EnvVar `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse GET response: %v", err)
	}
	vars := resp.Data

	// Find the variable we just created
	var found *model.EnvVar
	for i := range vars {
		if vars[i].Key == "DATABASE_HOST" {
			found = &vars[i]
			break
		}
	}

	if found == nil {
		t.Error("Posted variable not found in GET response")
		return
	}

	if found.Type != "string" {
		t.Errorf("Expected type string, got %s", found.Type)
	}
	if found.Value != "localhost" {
		t.Errorf("Expected value 'localhost', got %q", found.Value)
	}
}

// TestGetEnvLazyMigration tests lazy migration of legacy env vars
// This covers Domain 5 migration scenarios from the spec
func TestGetEnvLazyMigration(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*service.ConfigService) error
		expectedLen int
		checks      func(t *testing.T, vars []model.EnvVar)
	}{
		{
			name: "Legacy empty type migrates to string",
			setup: func(cs *service.ConfigService) error {
				cfg, _ := cs.Get()
				cfg.EnvVars = []model.EnvVar{
					{Key: "LEGACY_VAR", Value: "value", Type: ""},
				}
				return cs.Save(cfg)
			},
			expectedLen: 1,
			checks: func(t *testing.T, vars []model.EnvVar) {
				if vars[0].Type != "string" {
					t.Errorf("Expected type 'string', got %s", vars[0].Type)
				}
			},
		},
		{
			name: "Legacy plain type migrates to string",
			setup: func(cs *service.ConfigService) error {
				cfg, _ := cs.Get()
				cfg.EnvVars = []model.EnvVar{
					{Key: "PLAIN_VAR", Value: "value", Type: "plain"},
				}
				return cs.Save(cfg)
			},
			expectedLen: 1,
			checks: func(t *testing.T, vars []model.EnvVar) {
				if vars[0].Type != "string" {
					t.Errorf("Expected type 'string', got %s", vars[0].Type)
				}
			},
		},
		{
			name: "Encrypted variable without explicit type migrates to secret",
			setup: func(cs *service.ConfigService) error {
				cfg, _ := cs.Get()
				cfg.EnvVars = []model.EnvVar{
					{Key: "SECRET_VAR", Value: "encrypted_value", Type: "", Encrypted: true},
				}
				return cs.Save(cfg)
			},
			expectedLen: 1,
			checks: func(t *testing.T, vars []model.EnvVar) {
				if vars[0].Type != "secret" {
					t.Errorf("Expected type 'secret', got %s", vars[0].Type)
				}
				if !vars[0].Encrypted {
					t.Error("Expected Encrypted flag to be true")
				}
			},
		},
		{
			name: "Already-migrated vars remain unchanged",
			setup: func(cs *service.ConfigService) error {
				cfg, _ := cs.Get()
				cfg.EnvVars = []model.EnvVar{
					{Key: "STRING_VAR", Value: "value", Type: "string"},
					{Key: "NUMBER_VAR", Value: "42", Type: "number"},
					{Key: "BOOL_VAR", Value: "true", Type: "boolean"},
					{Key: "SECRET_VAR", Value: "secret", Type: "secret", Encrypted: true},
				}
				return cs.Save(cfg)
			},
			expectedLen: 4,
			checks: func(t *testing.T, vars []model.EnvVar) {
				types := map[string]string{}
				for _, v := range vars {
					types[v.Key] = v.Type
				}
				if types["STRING_VAR"] != "string" {
					t.Error("String var type changed")
				}
				if types["NUMBER_VAR"] != "number" {
					t.Error("Number var type changed")
				}
				if types["BOOL_VAR"] != "boolean" {
					t.Error("Boolean var type changed")
				}
				if types["SECRET_VAR"] != "secret" {
					t.Error("Secret var type changed")
				}
			},
		},
		{
			name: "Mixed legacy and new vars — only legacy ones migrate",
			setup: func(cs *service.ConfigService) error {
				cfg, _ := cs.Get()
				cfg.EnvVars = []model.EnvVar{
					{Key: "LEGACY", Value: "old", Type: ""},
					{Key: "MODERN", Value: "new", Type: "string"},
				}
				return cs.Save(cfg)
			},
			expectedLen: 2,
			checks: func(t *testing.T, vars []model.EnvVar) {
				for _, v := range vars {
					if v.Key == "LEGACY" && v.Type != "string" {
						t.Errorf("Legacy var not migrated, type = %s", v.Type)
					}
					if v.Key == "MODERN" && v.Type != "string" {
						t.Errorf("Modern var was modified, type = %s", v.Type)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			configSvc := service.NewConfigService(t.TempDir())
			if err := tt.setup(configSvc); err != nil {
				t.Fatalf("Failed to setup test: %v", err)
			}

			envSvc := service.NewEnvService(configSvc)
			handler := NewEnvHandler(envSvc, t.TempDir())

			// Call GetEnv which should trigger lazy migration
			req := httptest.NewRequest("GET", "/api/env", nil)
			w := httptest.NewRecorder()
			handler.GetEnv(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("GetEnv failed: %d — %s", w.Code, w.Body.String())
			}

			// Parse response
			var resp struct {
				Data []model.EnvVar `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}
			vars := resp.Data

			// Check count
			if len(vars) != tt.expectedLen {
				t.Errorf("Expected %d vars, got %d", tt.expectedLen, len(vars))
				return
			}

			// Run custom checks
			tt.checks(t, vars)

			// Verify migration was persisted
			saved, err := configSvc.Get()
			if err != nil {
				t.Fatalf("Failed to read saved config: %v", err)
			}

			for _, v := range saved.EnvVars {
				// No legacy types should remain
				if v.Type == "" || v.Type == "plain" {
					t.Errorf("Config still has legacy type for %s: %s", v.Key, v.Type)
				}
			}
		})
	}
}

// TestGetEnvMigrationIdempotence verifies that calling GetEnv multiple times
// after migration doesn't change the data or cause errors
func TestGetEnvMigrationIdempotence(t *testing.T) {
	// Setup config with legacy var
	configSvc := service.NewConfigService(t.TempDir())
	cfg, _ := configSvc.Get()
	cfg.EnvVars = []model.EnvVar{
		{Key: "LEGACY_VAR", Value: "legacy_value", Type: ""},
	}
	_ = configSvc.Save(cfg)

	envSvc := service.NewEnvService(configSvc)
	handler := NewEnvHandler(envSvc, t.TempDir())

	// First GET (triggers migration)
	req1 := httptest.NewRequest("GET", "/api/env", nil)
	w1 := httptest.NewRecorder()
	handler.GetEnv(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("First GetEnv failed: %d", w1.Code)
	}

	var resp1 struct {
		Data []model.EnvVar `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	vars1 := resp1.Data

	// Second GET (should return same result, migration should be idempotent)
	req2 := httptest.NewRequest("GET", "/api/env", nil)
	w2 := httptest.NewRecorder()
	handler.GetEnv(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Second GetEnv failed: %d", w2.Code)
	}

	var resp2 struct {
		Data []model.EnvVar `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	vars2 := resp2.Data

	// Compare results
	if len(vars1) != len(vars2) {
		t.Errorf("Response length changed between calls: %d vs %d", len(vars1), len(vars2))
		return
	}

	if len(vars1) == 0 {
		t.Error("No variables returned")
		return
	}

	if vars1[0].Type != vars2[0].Type {
		t.Errorf("Type changed between calls: %s vs %s", vars1[0].Type, vars2[0].Type)
	}

	if vars1[0].Value != vars2[0].Value {
		t.Errorf("Value changed between calls")
	}
}

// TestPostEnvUpdateExistingVariable tests updating an existing variable
// with different type and description
func TestPostEnvUpdateExistingVariable(t *testing.T) {
	configSvc := service.NewConfigService(t.TempDir())
	envSvc := service.NewEnvService(configSvc)
	handler := NewEnvHandler(envSvc, t.TempDir())

	// Create initial variable as string
	payload1 := map[string]interface{}{
		"key":   "MY_VAR",
		"value": "initial_value",
		"type":  "string",
	}
	body, _ := json.Marshal(payload1)
	req := httptest.NewRequest("POST", "/api/env", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.PostEnv(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Initial POST failed: %d", w.Code)
	}

	// Update to number type
	payload2 := map[string]interface{}{
		"key":         "MY_VAR",
		"value":       "42",
		"type":        "number",
		"description": "Updated to number type",
	}
	body, _ = json.Marshal(payload2)
	req = httptest.NewRequest("POST", "/api/env", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.PostEnv(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Update POST failed: %d", w.Code)
	}

	// Verify the update
	config, _ := configSvc.Get()
	var found *model.EnvVar
	for i := range config.EnvVars {
		if config.EnvVars[i].Key == "MY_VAR" {
			found = &config.EnvVars[i]
			break
		}
	}

	if found == nil {
		t.Error("Variable not found after update")
		return
	}

	if found.Type != "number" {
		t.Errorf("Expected type 'number', got %s", found.Type)
	}
	if found.Value != "42" {
		t.Errorf("Expected value '42', got %q", found.Value)
	}
	if found.Description != "Updated to number type" {
		t.Errorf("Description not updated, got %q", found.Description)
	}
}

// TestPostEnvSecretHandling tests special handling of secret type variables
func TestPostEnvSecretHandling(t *testing.T) {
	configSvc := service.NewConfigService(t.TempDir())
	envSvc := service.NewEnvService(configSvc)
	handler := NewEnvHandler(envSvc, t.TempDir())

	// Create a secret variable
	payload := map[string]interface{}{
		"key":   "API_KEY",
		"value": "secret-key-value",
		"type":  "secret",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/env", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.PostEnv(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to create secret: %d", w.Code)
	}

	// Verify Encrypted flag is set
	config, _ := configSvc.Get()
	var found *model.EnvVar
	for i := range config.EnvVars {
		if config.EnvVars[i].Key == "API_KEY" {
			found = &config.EnvVars[i]
			break
		}
	}

	if found == nil {
		t.Error("Secret variable not found")
		return
	}

	if !found.Encrypted {
		t.Error("Expected Encrypted flag to be true for secret type")
	}

	if found.Type != "secret" {
		t.Errorf("Expected type 'secret', got %s", found.Type)
	}
}
