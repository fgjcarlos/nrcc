package service

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

var managedEnvNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

type ManagedEnvService struct {
	dataDir string
}

func NewManagedEnvService(dataDir string) ManagedEnvService {
	return ManagedEnvService{dataDir: dataDir}
}

func (s ManagedEnvService) Load() (model.ManagedEnvState, error) {
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

	variables, err := parseManagedEnv(string(raw))
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
		return model.ManagedEnvState{
			Variables:       candidate,
			RestartRequired: true,
		}, nil
	}

	rendered := renderManagedEnv(candidate)
	if err := platform.WriteFileAtomic(filepath.Join(s.dataDir, ".env.managed"), []byte(rendered), 0o600); err != nil {
		return model.ManagedEnvState{}, fmt.Errorf("write managed env: %w", err)
	}

	return model.ManagedEnvState{
		Variables:       candidate,
		RestartRequired: true,
	}, nil
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
			Name:  name,
			Value: value,
		})
	}

	sort.SliceStable(variables, func(i, j int) bool {
		return variables[i].Name < variables[j].Name
	})

	return variables
}

func parseManagedEnv(raw string) ([]model.ManagedEnvVar, error) {
	lines := strings.Split(raw, "\n")
	var variables []model.ManagedEnvVar
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid managed env line: %q", line)
		}
		variables = append(variables, model.ManagedEnvVar{
			Name:  strings.TrimSpace(name),
			Value: strings.TrimSpace(value),
		})
	}
	return normalizeManagedEnv(variables), nil
}

func renderManagedEnv(variables []model.ManagedEnvVar) string {
	if len(variables) == 0 {
		return ""
	}

	lines := make([]string, 0, len(variables))
	for _, variable := range variables {
		lines = append(lines, fmt.Sprintf("%s=%s", variable.Name, variable.Value))
	}
	return strings.Join(lines, "\n") + "\n"
}
