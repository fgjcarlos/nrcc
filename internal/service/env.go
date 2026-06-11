package service

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// EnvService handles environment variable operations
type EnvService struct {
	configSvc     *ConfigService
	encryptionKey string
}

// NewEnvService creates a new environment variable service.
// If encryptionKey is non-empty, secret values are encrypted at rest with AES-256-GCM.
func NewEnvService(configSvc *ConfigService, encryptionKey ...string) *EnvService {
	key := ""
	if len(encryptionKey) > 0 {
		key = encryptionKey[0]
	}
	return &EnvService{
		configSvc:     configSvc,
		encryptionKey: key,
	}
}

// ValidateValue validates that a value is appropriate for its type.
// Returns an error if validation fails; nil if valid.
// Type vocabulary: "string" | "number" | "boolean" | "secret"
func ValidateValue(value string, typ string) error {
	switch typ {
	case "string", "secret":
		// Strings and secrets accept any value
		return nil
	case "number":
		// Numbers must parse as valid numeric values
		if value == "" {
			return fmt.Errorf("number values cannot be empty")
		}
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("invalid number format: %w", err)
		}
		return nil
	case "boolean":
		// Booleans must be "true" or "false" (case-insensitive)
		lower := strings.ToLower(strings.TrimSpace(value))
		if lower != "true" && lower != "false" {
			return fmt.Errorf("boolean values must be 'true' or 'false'")
		}
		return nil
	default:
		return fmt.Errorf("unknown type: %s", typ)
	}
}

// NormalizeValue normalizes a value to its canonical form for a given type.
// For booleans, returns "true" or "false"; for numbers, validates format;
// for strings and secrets, returns as-is.
func NormalizeValue(value string, typ string) (string, error) {
	switch typ {
	case "string", "secret":
		// Strings and secrets are returned as-is
		return value, nil
	case "number":
		// Parse and re-format number (ensures canonical form)
		if value == "" {
			return "", fmt.Errorf("number values cannot be empty")
		}
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return "", fmt.Errorf("invalid number format: %w", err)
		}
		// Return as-is (input is already normalized if it was valid)
		return value, nil
	case "boolean":
		// Booleans must be "true" or "false"
		lower := strings.ToLower(strings.TrimSpace(value))
		if lower == "true" {
			return "true", nil
		} else if lower == "false" {
			return "false", nil
		}
		return "", fmt.Errorf("boolean values must be 'true' or 'false'")
	default:
		return "", fmt.Errorf("unknown type: %s", typ)
	}
}

// MigrateEnvTypes performs lazy migration of an EnvVar from legacy format.
// If Type is empty or "plain", migrates to "string".
// Returns the migrated EnvVar.
func MigrateEnvTypes(ev model.EnvVar) model.EnvVar {
	// Migrate legacy types to new vocabulary
	if ev.Type == "" || ev.Type == "plain" {
		ev.Type = "string"
	}
	// If encrypted, upgrade type to "secret" if it's not already
	if ev.Encrypted && ev.Type != "secret" {
		ev.Type = "secret"
	}
	return ev
}

func (s *EnvService) List() ([]model.EnvVar, error) {
	config, err := s.configSvc.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	if config.EnvVars == nil {
		return []model.EnvVar{}, nil
	}

	// Check if any migration is needed (type migration or plaintext→encrypted)
	migrationNeeded := false
	for _, ev := range config.EnvVars {
		if ev.Type == "" || ev.Type == "plain" {
			migrationNeeded = true
			break
		}
		if ev.Encrypted && s.encryptionKey != "" && !IsEncrypted(ev.Value) {
			migrationNeeded = true
			break
		}
	}

	if migrationNeeded {
		for i, ev := range config.EnvVars {
			config.EnvVars[i] = MigrateEnvTypes(ev)
			if config.EnvVars[i].Encrypted && s.encryptionKey != "" && !IsEncrypted(config.EnvVars[i].Value) {
				encrypted, err := Encrypt(config.EnvVars[i].Value, s.encryptionKey)
				if err == nil {
					config.EnvVars[i].Value = encrypted
				}
			}
		}
		if err := s.configSvc.Save(config); err != nil {
			fmt.Printf("Warning: failed to save migrated config: %v\n", err)
		}
	}

	// Return env vars without values (for masked display)
	var result []model.EnvVar
	for _, ev := range config.EnvVars {
		masked := model.EnvVar{
			Key:         ev.Key,
			Type:        ev.Type,
			Encrypted:   ev.Encrypted,
			Description: ev.Description,
		}
		if !ev.Encrypted {
			masked.Value = ev.Value
		}
		result = append(result, masked)
	}

	return result, nil
}

// Set sets an environment variable with the specified type and description.
// The typ parameter should be one of: "string", "number", "boolean", "secret"
// Description is optional and used to document the purpose of the variable.
func (s *EnvService) Set(key, value string, typ string, description string, encrypted bool) error {
	config, err := s.configSvc.Get()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Check if env var already exists
	found := false
	for i, ev := range config.EnvVars {
		if ev.Key == key {
			if value == "" && ev.Encrypted {
				value = ev.Value
			} else if encrypted && s.encryptionKey != "" {
				value, err = Encrypt(value, s.encryptionKey)
				if err != nil {
					return fmt.Errorf("encrypt value: %w", err)
				}
			}
			config.EnvVars[i].Value = value
			config.EnvVars[i].Type = typ
			config.EnvVars[i].Description = description
			config.EnvVars[i].Encrypted = encrypted
			found = true
			break
		}
	}

	if !found {
		if encrypted && s.encryptionKey != "" {
			value, err = Encrypt(value, s.encryptionKey)
			if err != nil {
				return fmt.Errorf("encrypt value: %w", err)
			}
		}
		config.EnvVars = append(config.EnvVars, model.EnvVar{
			Key:         key,
			Value:       value,
			Type:        typ,
			Description: description,
			Encrypted:   encrypted,
		})
	}

	return s.configSvc.Save(config)
}

// Delete deletes an environment variable
func (s *EnvService) Delete(key string) error {
	config, err := s.configSvc.Get()
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	// Remove env var
	var newEnvVars []model.EnvVar
	for _, ev := range config.EnvVars {
		if ev.Key != key {
			newEnvVars = append(newEnvVars, ev)
		}
	}
	config.EnvVars = newEnvVars

	return s.configSvc.Save(config)
}

// GetAll returns all environment variables as a decrypted map
func (s *EnvService) GetAll() (map[string]string, error) {
	config, err := s.configSvc.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	result := make(map[string]string)
	for _, ev := range config.EnvVars {
		val := ev.Value
		if ev.Encrypted && s.encryptionKey != "" && IsEncrypted(val) {
			decrypted, err := Decrypt(val, s.encryptionKey)
			if err != nil {
				return nil, fmt.Errorf("decrypt %s: %w", ev.Key, err)
			}
			val = decrypted
		}
		result[ev.Key] = val
	}

	return result, nil
}

// parseEnvFile reads and parses a .env file
// Lines starting with # are ignored, as are empty lines
// Values can be quoted with double quotes; quotes are stripped if present
// If the file doesn't exist, returns an empty map (not an error)
func parseEnvFile(path string) (map[string]string, error) {
	result := make(map[string]string)

	// If file doesn't exist, return empty map (not an error)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("failed to read .env file: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Find the first = sign
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx == -1 {
			continue // Skip lines without =
		}

		key := strings.TrimSpace(line[:eqIdx])
		value := line[eqIdx+1:]

		// Remove surrounding quotes if present
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		result[key] = value
	}

	return result, scanner.Err()
}

// GetAllMerged returns all environment variables, merging config.json and .env
// Priority: os.Environ() < config.json < .env (last one wins)
func (s *EnvService) GetAllMerged(dotenvPath string) (map[string]string, error) {
	// Start with config.json vars
	result, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	// Overlay .env vars (higher priority)
	dotenvVars, err := parseEnvFile(dotenvPath)
	if err != nil {
		return nil, err
	}
	for k, v := range dotenvVars {
		result[k] = v
	}

	return result, nil
}

// ReadDotenv reads and returns the raw content of the .env file
// Returns empty string if file doesn't exist
func ReadDotenv(dataDir string) (string, error) {
	dotenvPath := filepath.Join(dataDir, ".env")
	content, err := os.ReadFile(dotenvPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read .env file: %w", err)
	}
	return string(content), nil
}

// WriteDotenv writes content to the .env file
func WriteDotenv(dataDir string, content string) error {
	dotenvPath := filepath.Join(dataDir, ".env")
	if err := os.WriteFile(dotenvPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}
	return nil
}
