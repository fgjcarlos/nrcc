package service

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

var packageNamePattern = regexp.MustCompile(`^(?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*$`)

type commandRunner interface {
	Run(dir string, name string, args ...string) (string, error)
}

type LibraryService struct {
	dataDir string
	runner  commandRunner
}

func NewLibraryService(dataDir string) LibraryService {
	runner := platform.NewRunner()
	runner.Timeout = 2 * time.Minute
	return LibraryService{
		dataDir: dataDir,
		runner:  runner,
	}
}

func (s LibraryService) List() (model.LibraryList, error) {
	output, err := s.runner.Run(s.dataDir, "npm", "ls", "--json", "--depth=0")
	if err != nil {
		return model.LibraryList{}, fmt.Errorf("list npm libraries: %w", err)
	}
	items, err := parseLibraryList(output)
	if err != nil {
		return model.LibraryList{}, err
	}
	return model.LibraryList{Items: items}, nil
}

func (s LibraryService) Install(name string) (model.LibraryOperationResult, error) {
	name, err := validatePackageName(name)
	if err != nil {
		return model.LibraryOperationResult{}, err
	}

	output, err := s.runner.Run(s.dataDir, "npm", "install", name)
	if err != nil {
		return model.LibraryOperationResult{}, fmt.Errorf("install npm library: %w", err)
	}

	pkg, err := s.lookupInstalledPackage(name)
	if err != nil {
		return model.LibraryOperationResult{}, err
	}

	return model.LibraryOperationResult{
		Package:   pkg,
		Message:   fmt.Sprintf("%s installed successfully", name),
		Output:    output,
		Operation: "install",
	}, nil
}

func (s LibraryService) Uninstall(name string) (model.LibraryOperationResult, error) {
	name, err := validatePackageName(name)
	if err != nil {
		return model.LibraryOperationResult{}, err
	}

	output, err := s.runner.Run(s.dataDir, "npm", "uninstall", name)
	if err != nil {
		return model.LibraryOperationResult{}, fmt.Errorf("uninstall npm library: %w", err)
	}

	return model.LibraryOperationResult{
		Package: model.LibraryPackage{
			Name:   name,
			Direct: true,
		},
		Message:   fmt.Sprintf("%s removed successfully", name),
		Output:    output,
		Operation: "uninstall",
	}, nil
}

func (s LibraryService) lookupInstalledPackage(name string) (model.LibraryPackage, error) {
	list, err := s.List()
	if err != nil {
		return model.LibraryPackage{}, err
	}

	for _, item := range list.Items {
		if item.Name == name {
			return item, nil
		}
	}

	return model.LibraryPackage{
		Name:   name,
		Direct: true,
	}, nil
}

func validatePackageName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("package name is required")
	}
	if strings.ContainsAny(name, " \t\n\r") {
		return "", fmt.Errorf("package name must not contain whitespace")
	}
	if strings.Contains(name, "/../") || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("package name must not be a path")
	}
	if strings.HasPrefix(name, "file:") || strings.HasPrefix(name, "git+") || strings.Contains(name, "://") {
		return "", fmt.Errorf("package name must be a registry package name")
	}
	if strings.HasPrefix(name, "-") {
		return "", fmt.Errorf("package name must not be a flag")
	}
	if !packageNamePattern.MatchString(name) {
		return "", fmt.Errorf("package name is invalid")
	}
	return name, nil
}

func parseLibraryList(raw string) ([]model.LibraryPackage, error) {
	type dependency struct {
		Version string `json:"version"`
	}
	type npmList struct {
		Dependencies map[string]dependency `json:"dependencies"`
	}

	var payload npmList
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("parse npm library list: %w", err)
	}

	items := make([]model.LibraryPackage, 0, len(payload.Dependencies))
	for name, dep := range payload.Dependencies {
		items = append(items, model.LibraryPackage{
			Name:    name,
			Version: strings.TrimSpace(dep.Version),
			Direct:  true,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func runtimePackageLockPath(dataDir string) string {
	return filepath.Join(dataDir, "package-lock.json")
}
