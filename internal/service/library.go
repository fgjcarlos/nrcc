package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/composedof2/nrcc/internal/model"
)

// LibraryService handles npm library operations
type LibraryService struct {
	dataDir string
}

// NewLibraryService creates a new library service
func NewLibraryService(dataDir string) *LibraryService {
	return &LibraryService{
		dataDir: dataDir,
	}
}

// List returns installed npm packages
func (s *LibraryService) List() ([]model.LibraryInfo, error) {
	pkgPath := filepath.Join(s.dataDir, "package.json")

	data, err := os.ReadFile(pkgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.LibraryInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	deps, ok := pkg["dependencies"].(map[string]interface{})
	if !ok {
		return []model.LibraryInfo{}, nil
	}

	var libraries []model.LibraryInfo
	for name, version := range deps {
		if verStr, ok := version.(string); ok {
			libraries = append(libraries, model.LibraryInfo{
				Name:    name,
				Version: verStr,
			})
		}
	}

	return libraries, nil
}

// Install installs an npm package
func (s *LibraryService) Install(pkg string) error {
	cmd := exec.Command("npm", "install", pkg)
	cmd.Dir = s.dataDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	return nil
}

// Uninstall uninstalls an npm package
func (s *LibraryService) Uninstall(pkg string) error {
	cmd := exec.Command("npm", "uninstall", pkg)
	cmd.Dir = s.dataDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall package: %w", err)
	}

	return nil
}

// Search searches npm registry for packages
func (s *LibraryService) Search(query string) ([]interface{}, error) {
	cmd := exec.Command("npm", "search", "--json", query)
	cmd.Dir = s.dataDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}

	// Parse JSON output
	var results []interface{}
	if err := json.Unmarshal(output, &results); err != nil {
		// npm search sometimes returns errors; return empty result
		return []interface{}{}, nil
	}

	return results, nil
}

// Check checks if a package is available
func (s *LibraryService) Check(pkg string) (bool, error) {
	cmd := exec.Command("npm", "view", pkg)
	cmd.Dir = s.dataDir
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		return false, nil // Package not found
	}

	return true, nil
}
