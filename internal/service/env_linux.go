//go:build linux

package service

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// EnvService handles environment variable operations
type EnvService struct {
	configSvc          *ConfigService
	encryptionKey      string
	mu                 sync.Mutex
	pm                 *ProcessManager
	syncNodeRedGlobals func([]string) error
}

// NewEnvService creates a new environment variable service.
// If encryptionKey is non-empty, secret values are encrypted at rest with AES-256-GCM.
func NewEnvService(configSvc *ConfigService, encryptionKey ...string) *EnvService {
	key := ""
	if len(encryptionKey) > 0 {
		key = encryptionKey[0]
	}
	s := &EnvService{
		configSvc:     configSvc,
		encryptionKey: key,
	}
	s.syncNodeRedGlobals = s.syncNodeRedGlobalEnvs
	return s
}

// EncryptionKeyConfigured reports whether NRCC_ENCRYPTION_KEY is set
// non-empty. Used by the security posture endpoint (issue #676 item 2)
// to flag the silent-degradation failure mode where Encrypted env vars
// are written in clear when the key is missing.
func (s *EnvService) EncryptionKeyConfigured() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.encryptionKey != ""
}

// SetProcessManager records the active ProcessManager so env sync can reuse
// the same runtime contract as ProcessManager.Start.
func (s *EnvService) SetProcessManager(pm *ProcessManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pm = pm
}

func ValidateEnvKey(key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if strings.ContainsAny(key, "\x00\r\n=") {
		return fmt.Errorf("key cannot contain NUL, newline, or '='")
	}
	return nil
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
		switch lower {
		case "true":
			return "true", nil
		case "false":
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
	for {
		config, err := s.configSvc.Get()
		if err != nil {
			return nil, fmt.Errorf("failed to get config: %w", err)
		}
		if !envMigrationNeeded(config.EnvVars, s.encryptionKey) {
			return maskEnvVars(config.EnvVars), nil
		}

		staged, err := stageEnvMigration(config.EnvVars, s.encryptionKey)
		if err != nil {
			return nil, err
		}
		committed, err := s.configSvc.Update(func(current *model.NodeRedConfig) error {
			if !envMigrationNeeded(current.EnvVars, s.encryptionKey) {
				return errEnvMigrationAlreadyDone
			}
			for i, envVar := range current.EnvVars {
				migrated := MigrateEnvTypes(envVar)
				if migrated.Encrypted && s.encryptionKey != "" && !IsEncrypted(migrated.Value) {
					encrypted, ok := staged[envMigrationKey{key: migrated.Key, value: migrated.Value}]
					if !ok {
						return errEnvMigrationStale
					}
					migrated.Value = encrypted
				}
				current.EnvVars[i] = migrated
			}
			return nil
		})
		if errors.Is(err, errEnvMigrationStale) {
			continue
		}
		if errors.Is(err, errEnvMigrationAlreadyDone) {
			current, getErr := s.configSvc.Get()
			if getErr != nil {
				return nil, fmt.Errorf("read migrated environment config: %w", getErr)
			}
			return maskEnvVars(current.EnvVars), nil
		}
		if err != nil {
			return nil, fmt.Errorf("migrate environment config: %w", err)
		}
		return maskEnvVars(committed.EnvVars), nil
	}
}

var errEnvMigrationStale = errors.New("environment changed while staging migration")
var errEnvMigrationAlreadyDone = errors.New("environment migration already committed")

// ErrEncryptionKeyRequired is returned when a caller attempts to persist an
// encrypted env var while NRCC_ENCRYPTION_KEY is empty. The store would
// otherwise silently write the secret in clear — the exact silent
// degradation failure mode that the SecurityPostureCard surfaces.
var ErrEncryptionKeyRequired = errors.New("NRCC_ENCRYPTION_KEY is not configured; cannot store an encrypted value")

type envMigrationKey struct {
	key   string
	value string
}

func envMigrationNeeded(envVars []model.EnvVar, encryptionKey string) bool {
	for _, envVar := range envVars {
		if envVar.Type == "" || envVar.Type == "plain" || envVar.Encrypted && envVar.Type != "secret" ||
			envVar.Encrypted && encryptionKey != "" && !IsEncrypted(envVar.Value) {
			return true
		}
	}
	return false
}

func stageEnvMigration(envVars []model.EnvVar, encryptionKey string) (map[envMigrationKey]string, error) {
	staged := make(map[envMigrationKey]string)
	for _, envVar := range envVars {
		if !envVar.Encrypted || encryptionKey == "" || IsEncrypted(envVar.Value) {
			continue
		}
		encrypted, err := Encrypt(envVar.Value, encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt migrated environment variable %s: %w", envVar.Key, err)
		}
		staged[envMigrationKey{key: envVar.Key, value: envVar.Value}] = encrypted
	}
	return staged, nil
}

func maskEnvVars(envVars []model.EnvVar) []model.EnvVar {
	if len(envVars) == 0 {
		return []model.EnvVar{}
	}
	var result []model.EnvVar
	for _, ev := range envVars {
		masked := model.EnvVar{
			Key:         ev.Key,
			Type:        ev.Type,
			Encrypted:   ev.Encrypted,
			Description: ev.Description,
			Source:      envVarSource(ev),
		}
		if !ev.Encrypted {
			masked.Value = ev.Value
		}
		result = append(result, masked)
	}

	return result
}

// Set sets an environment variable with the specified type and description.
// The typ parameter should be one of: "string", "number", "boolean", "secret"
// Description is optional and used to document the purpose of the variable.
func (s *EnvService) Set(key, value string, typ string, description string, encrypted bool) error {
	return s.set(key, value, typ, description, encrypted, "nrcc", true)
}

func envVarSource(envVar model.EnvVar) string {
	if envVar.Source == "node-red" {
		return "node-red"
	}
	return "nrcc"
}

func (s *EnvService) set(key, value string, typ string, description string, encrypted bool, source string, syncGlobal bool) error {
	if err := ValidateEnvKey(key); err != nil {
		return err
	}
	// #664: fail-closed when the caller wants to store an encrypted value
	// but the operator has not configured NRCC_ENCRYPTION_KEY. Persisting
	// the plaintext value would still be flagged Encrypted: true in
	// config.json and propagate into every backup, so reject the write
	// before it can hit disk. An empty value is allowed (preserves the
	// existing encrypted entry unchanged — see the committedValue branch
	// below) because no new plaintext would be written.
	if encrypted && value != "" && s.encryptionKey == "" {
		return ErrEncryptionKeyRequired
	}
	stagedValue := value
	var err error
	if encrypted && value != "" && s.encryptionKey != "" {
		stagedValue, err = Encrypt(value, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("encrypt value: %w", err)
		}
	}
	previousNodeRedGlobal := false
	_, err = s.configSvc.Update(func(current *model.NodeRedConfig) error {
		found := false
		for i, envVar := range current.EnvVars {
			if envVar.Key != key {
				continue
			}
			previousNodeRedGlobal = !envVar.Encrypted && envVar.Type != "secret"
			committedValue := stagedValue
			if value == "" && envVar.Encrypted {
				committedValue = envVar.Value
			}
			current.EnvVars[i] = model.EnvVar{
				Key:         key,
				Value:       committedValue,
				Type:        typ,
				Description: description,
				Encrypted:   encrypted,
				Source:      source,
			}
			found = true
			break
		}
		if !found {
			current.EnvVars = append(current.EnvVars, model.EnvVar{
				Key:         key,
				Value:       stagedValue,
				Type:        typ,
				Description: description,
				Encrypted:   encrypted,
				Source:      source,
			})
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update environment config: %w", err)
	}
	if syncGlobal && (previousNodeRedGlobal || !encrypted && typ != "secret") {
		s.mu.Lock()
		defer s.mu.Unlock()
		nodeRedVar, err := s.currentNodeRedGlobal(key)
		if err != nil {
			return fmt.Errorf("environment JSON committed but current value reload failed: %w", err)
		}
		if err := s.syncNodeRedGlobalEnv(key, nodeRedVar); err != nil {
			return fmt.Errorf("environment JSON committed but Node-RED global sync failed: %w", err)
		}
	}
	return nil
}

// Delete deletes an environment variable
func (s *EnvService) Delete(key string) error {
	if err := ValidateEnvKey(key); err != nil {
		return err
	}
	managed := false
	_, err := s.configSvc.Update(func(current *model.NodeRedConfig) error {
		newEnvVars := make([]model.EnvVar, 0, len(current.EnvVars))
		for _, envVar := range current.EnvVars {
			if envVar.Key != key {
				newEnvVars = append(newEnvVars, envVar)
			} else {
				managed = !envVar.Encrypted && envVar.Type != "secret"
			}
		}
		current.EnvVars = newEnvVars
		return nil
	})
	if err != nil {
		return fmt.Errorf("update environment config: %w", err)
	}
	if managed {
		s.mu.Lock()
		defer s.mu.Unlock()
		nodeRedVar, err := s.currentNodeRedGlobal(key)
		if err != nil {
			return fmt.Errorf("environment JSON committed but current value reload failed: %w", err)
		}
		if err := s.syncNodeRedGlobalEnv(key, nodeRedVar); err != nil {
			return fmt.Errorf("environment JSON committed but Node-RED global sync failed: %w", err)
		}
	}
	return nil
}

func (s *EnvService) currentNodeRedGlobal(key string) (*model.EnvVar, error) {
	config, err := s.configSvc.Get()
	if err != nil {
		return nil, err
	}
	for _, envVar := range config.EnvVars {
		if envVar.Key == key && !envVar.Encrypted && envVar.Type != "secret" {
			current := envVar
			return &current, nil
		}
	}
	return nil, nil
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
	// #nosec G304 -- path is built from operator-supplied dataDir + a constant filename; not request-derived.
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
	// #nosec G304 -- dotenvPath is built from operator-supplied dataDir + a constant filename; not request-derived.
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
