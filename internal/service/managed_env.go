package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

var managedEnvNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

const (
	managedEnvEncryptedPrefix = "nrcc:enc:v1:"
	managedEnvKeyFile         = ".env.managed.key"
)

type ManagedEnvService struct {
	dataDir string
}

func NewManagedEnvService(dataDir string) ManagedEnvService {
	return ManagedEnvService{dataDir: dataDir}
}

func (s ManagedEnvService) Load() (model.ManagedEnvState, error) {
	state, err := s.loadActual()
	if err != nil {
		return model.ManagedEnvState{}, err
	}
	return maskManagedEnvState(state), nil
}

func (s ManagedEnvService) RuntimeLines() ([]string, error) {
	state, err := s.loadActual()
	if err != nil {
		return nil, err
	}

	lines := make([]string, 0, len(state.Variables))
	for _, variable := range state.Variables {
		lines = append(lines, fmt.Sprintf("%s=%s", variable.Name, variable.Value))
	}
	return lines, nil
}

func (s ManagedEnvService) loadActual() (model.ManagedEnvState, error) {
	path := filepath.Join(s.dataDir, ".env.managed")
	if !platform.Exists(path) {
		return model.ManagedEnvState{
			Variables:       []model.ManagedEnvVar{},
			RestartRequired: false,
		}, nil
	}

	raw, err := platform.ReadFile(path)
	if err != nil {
		return model.ManagedEnvState{}, fmt.Errorf("read managed env: %w", err)
	}

	variables, err := parseManagedEnv(s.dataDir, string(raw))
	if err != nil {
		return model.ManagedEnvState{}, err
	}

	return model.ManagedEnvState{
		Variables:       variables,
		RestartRequired: len(variables) > 0,
	}, nil
}

func (s ManagedEnvService) Validate(candidate []model.ManagedEnvVar) []string {
	var errors []string
	seen := map[string]struct{}{}

	for _, variable := range normalizeManagedEnv(candidate) {
		if variable.Name == "" {
			errors = append(errors, "environment variable name is required")
			continue
		}
		if strings.HasPrefix(variable.Name, "NRCC_") {
			errors = append(errors, fmt.Sprintf("%s is reserved for the control center", variable.Name))
		}
		if variable.Name == "PORT" {
			errors = append(errors, "PORT is reserved for the runtime process")
		}
		if !managedEnvNamePattern.MatchString(variable.Name) {
			errors = append(errors, fmt.Sprintf("%s must match ^[A-Z_][A-Z0-9_]*$", variable.Name))
		}
		if _, ok := seen[variable.Name]; ok {
			errors = append(errors, fmt.Sprintf("%s is duplicated", variable.Name))
			continue
		}
		seen[variable.Name] = struct{}{}
		if strings.ContainsRune(variable.Value, '\n') || strings.ContainsRune(variable.Value, '\r') {
			errors = append(errors, fmt.Sprintf("%s contains a newline", variable.Name))
		}
	}

	return errors
}

func (s ManagedEnvService) Apply(candidate []model.ManagedEnvVar) (model.ManagedEnvState, error) {
	candidate = normalizeManagedEnv(candidate)
	if errors := s.Validate(candidate); len(errors) > 0 {
		return maskManagedEnvState(model.ManagedEnvState{
			Variables:       candidate,
			RestartRequired: true,
		}), nil
	}

	existing, err := s.loadActual()
	if err != nil {
		return model.ManagedEnvState{}, err
	}

	rendered, err := s.renderManagedEnv(candidate, existing.Variables)
	if err != nil {
		return model.ManagedEnvState{}, err
	}
	if err := platform.WriteFileAtomic(filepath.Join(s.dataDir, ".env.managed"), []byte(rendered), 0o600); err != nil {
		return model.ManagedEnvState{}, fmt.Errorf("write managed env: %w", err)
	}

	return maskManagedEnvState(model.ManagedEnvState{
		Variables:       candidate,
		RestartRequired: true,
	}), nil
}

func (s ManagedEnvService) renderManagedEnv(candidate []model.ManagedEnvVar, existing []model.ManagedEnvVar) (string, error) {
	key, err := s.loadOrCreateKey(hasManagedEnvSecrets(candidate))
	if err != nil {
		return "", err
	}

	if len(candidate) == 0 {
		return "", nil
	}

	existingByName := make(map[string]model.ManagedEnvVar, len(existing))
	for _, variable := range existing {
		existingByName[variable.Name] = variable
	}

	lines := make([]string, 0, len(candidate))
	for _, variable := range candidate {
		value := variable.Value
		if variable.Secret {
			if value == "" && variable.HasValue {
				if existingVar, ok := existingByName[variable.Name]; ok && existingVar.Secret {
					value = existingVar.Value
				}
			}

			encrypted, err := encryptManagedEnvValue(key, value)
			if err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("%s=%s", variable.Name, encrypted))
			continue
		}

		lines = append(lines, fmt.Sprintf("%s=%s", variable.Name, value))
	}

	return strings.Join(lines, "\n") + "\n", nil
}

func normalizeManagedEnv(candidate []model.ManagedEnvVar) []model.ManagedEnvVar {
	variables := make([]model.ManagedEnvVar, 0, len(candidate))
	for _, variable := range candidate {
		name := strings.ToUpper(strings.TrimSpace(variable.Name))
		value := strings.TrimSpace(variable.Value)
		if name == "" && value == "" {
			continue
		}
		variables = append(variables, model.ManagedEnvVar{
			Name:     name,
			Value:    value,
			Secret:   variable.Secret,
			HasValue: variable.Secret && variable.HasValue,
		})
	}

	sort.SliceStable(variables, func(i, j int) bool {
		return variables[i].Name < variables[j].Name
	})

	return variables
}

func parseManagedEnv(dataDir string, raw string) ([]model.ManagedEnvVar, error) {
	lines := strings.Split(raw, "\n")
	var variables []model.ManagedEnvVar
	var key []byte
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid managed env line: %q", line)
		}

		secret := false
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, managedEnvEncryptedPrefix) {
			if key == nil {
				var err error
				key, err = loadManagedEnvKey(dataDir)
				if err != nil {
					return nil, err
				}
			}
			decrypted, err := decryptManagedEnvValue(key, value)
			if err != nil {
				return nil, err
			}
			value = decrypted
			secret = true
		}

		variables = append(variables, model.ManagedEnvVar{
			Name:     strings.TrimSpace(name),
			Value:    value,
			Secret:   secret,
			HasValue: secret,
		})
	}
	return normalizeManagedEnv(variables), nil
}

func hasManagedEnvSecrets(variables []model.ManagedEnvVar) bool {
	for _, variable := range variables {
		if variable.Secret {
			return true
		}
	}
	return false
}

func maskManagedEnvState(state model.ManagedEnvState) model.ManagedEnvState {
	masked := model.ManagedEnvState{
		Variables:       make([]model.ManagedEnvVar, 0, len(state.Variables)),
		RestartRequired: state.RestartRequired,
	}
	for _, variable := range state.Variables {
		if variable.Secret {
			masked.Variables = append(masked.Variables, model.ManagedEnvVar{
				Name:     variable.Name,
				Value:    "",
				Secret:   true,
				HasValue: true,
			})
			continue
		}
		masked.Variables = append(masked.Variables, model.ManagedEnvVar{
			Name:  variable.Name,
			Value: variable.Value,
		})
	}
	return masked
}

func (s ManagedEnvService) loadOrCreateKey(create bool) ([]byte, error) {
	if !create && !platform.Exists(filepath.Join(s.dataDir, managedEnvKeyFile)) {
		return nil, nil
	}

	if !platform.Exists(filepath.Join(s.dataDir, managedEnvKeyFile)) {
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("generate managed env key: %w", err)
		}
		encoded := base64.StdEncoding.EncodeToString(key)
		if err := platform.WriteFileAtomic(filepath.Join(s.dataDir, managedEnvKeyFile), []byte(encoded+"\n"), 0o600); err != nil {
			return nil, fmt.Errorf("write managed env key: %w", err)
		}
		return key, nil
	}

	return loadManagedEnvKey(s.dataDir)
}

func loadManagedEnvKey(dataDir string) ([]byte, error) {
	raw, err := platform.ReadFile(filepath.Join(dataDir, managedEnvKeyFile))
	if err != nil {
		return nil, fmt.Errorf("read managed env key: %w", err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("decode managed env key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("managed env key must be 32 bytes")
	}
	return key, nil
}

func encryptManagedEnvValue(key []byte, value string) (string, error) {
	if len(key) == 0 {
		return "", fmt.Errorf("managed env encryption key is not available")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create managed env cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create managed env gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate managed env nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, []byte(value), nil)
	combined := append(nonce, sealed...)
	return managedEnvEncryptedPrefix + base64.StdEncoding.EncodeToString(combined), nil
}

func decryptManagedEnvValue(key []byte, value string) (string, error) {
	if len(key) == 0 {
		return "", fmt.Errorf("managed env encryption key is not available")
	}
	encoded := strings.TrimPrefix(value, managedEnvEncryptedPrefix)
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode managed env secret: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create managed env cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create managed env gcm: %w", err)
	}
	if len(combined) < gcm.NonceSize() {
		return "", fmt.Errorf("managed env secret is truncated")
	}
	nonce := combined[:gcm.NonceSize()]
	ciphertext := combined[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt managed env secret: %w", err)
	}
	return string(plain), nil
}
