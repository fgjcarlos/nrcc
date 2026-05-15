package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/composedof2/nrcc/internal/model"
)

// LibraryService handles package library operations with pnpm
type LibraryService struct {
	dataDir string
	pm      PackageManager
}

// NewLibraryService creates a new library service with default pnpm package manager
func NewLibraryService(dataDir string) *LibraryService {
	return NewLibraryServiceWithPackageManager(dataDir, NewPnpmPackageManager(dataDir))
}

// NewLibraryServiceWithPackageManager creates a new library service with a custom package manager
func NewLibraryServiceWithPackageManager(dataDir string, pm PackageManager) *LibraryService {
	return &LibraryService{
		dataDir: dataDir,
		pm:      pm,
	}
}

// List returns installed npm packages with metadata from node_modules
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
			lib := model.LibraryInfo{
				Name:    name,
				Version: verStr,
			}

			// Enrich with metadata from node_modules
			if err := s.enrichLibraryMetadata(&lib); err != nil {
				// Log error but continue; metadata is optional
				// fmt.Printf("warning: failed to enrich metadata for %s: %v\n", name, err)
			}

			libraries = append(libraries, lib)
		}
	}

	return libraries, nil
}

// enrichLibraryMetadata reads package.json from node_modules to extract metadata
func (s *LibraryService) enrichLibraryMetadata(lib *model.LibraryInfo) error {
	pkgJSONPath := filepath.Join(s.dataDir, "node_modules", lib.Name, "package.json")

	data, err := os.ReadFile(pkgJSONPath)
	if err != nil {
		// Not all packages have package.json or may not be installed yet
		return nil
	}

	var pkg struct {
		Description string      `json:"description"`
		Keywords    []string    `json:"keywords"`
		Homepage    string      `json:"homepage"`
		Repository  interface{} `json:"repository"` // Can be string or object
		Author      string      `json:"author"`
		License     string      `json:"license"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	lib.Description = pkg.Description
	lib.Keywords = pkg.Keywords
	lib.Homepage = pkg.Homepage
	lib.Author = pkg.Author
	lib.License = pkg.License

	// Handle repository field — can be string or object with "url" property
	lib.Repository = extractRepositoryURL(pkg.Repository)

	return nil
}

// extractRepositoryURL handles both string and object repository formats
func extractRepositoryURL(repo interface{}) string {
	switch v := repo.(type) {
	case string:
		return v
	case map[string]interface{}:
		if url, ok := v["url"].(string); ok {
			return url
		}
	}
	return ""
}

// Install installs a package using pnpm add
func (s *LibraryService) Install(pkg string) error {
	return s.pm.Install(pkg)
}

// Uninstall uninstalls a package using pnpm remove
func (s *LibraryService) Uninstall(pkg string) error {
	return s.pm.Uninstall(pkg)
}

// searchRegistry represents the npm registry search response structure
type searchRegistry struct {
	Objects []struct {
		Package struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Date        string `json:"date"`
		} `json:"package"`
	} `json:"objects"`
}

// Search searches npm registry for packages via HTTP API
func (s *LibraryService) Search(query string) ([]interface{}, error) {
	// Build registry URL with query parameters
	registryURL := "https://registry.npmjs.org/-/v1/search"
	params := url.Values{}
	params.Set("text", query)
	params.Set("size", "10")

	fullURL := registryURL + "?" + params.Encode()

	// Make HTTP GET request to npm registry
	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search npm registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("npm registry returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var result searchRegistry
	if err := json.Unmarshal(body, &result); err != nil {
		// Return empty result on parse error
		return []interface{}{}, nil
	}

	// Convert result to []interface{} for backward compatibility
	results := make([]interface{}, len(result.Objects))
	for i, obj := range result.Objects {
		results[i] = map[string]interface{}{
			"name":        obj.Package.Name,
			"version":     obj.Package.Version,
			"description": obj.Package.Description,
			"date":        obj.Package.Date,
		}
	}

	return results, nil
}

// Check checks if a package is available using pnpm view
func (s *LibraryService) Check(pkg string) (bool, error) {
	cmd := exec.Command("pnpm", "view", pkg)
	cmd.Dir = s.dataDir
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		return false, nil // Package not found
	}

	return true, nil
}
