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
		Category    string      `json:"category"`
		Homepage    string      `json:"homepage"`
		Repository  interface{} `json:"repository"` // Can be string or object
		Author      interface{} `json:"author"`     // Can be string or object
		License     string      `json:"license"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	lib.Description = pkg.Description
	lib.Keywords = pkg.Keywords
	lib.Category = pkg.Category
	lib.Homepage = pkg.Homepage
	lib.Author = extractAuthorName(pkg.Author)
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

// extractAuthorName handles both string and object author formats.
func extractAuthorName(author interface{}) string {
	switch v := author.(type) {
	case string:
		return v
	case map[string]interface{}:
		if name, ok := v["name"].(string); ok {
			return name
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
		Downloads struct {
			Weekly int `json:"weekly"`
		} `json:"downloads"`
		Package struct {
			Name        string   `json:"name"`
			Version     string   `json:"version"`
			Description string   `json:"description"`
			Date        string   `json:"date"`
			Keywords    []string `json:"keywords"`
			Links       struct {
				NPM        string `json:"npm"`
				Homepage   string `json:"homepage"`
				Repository string `json:"repository"`
			} `json:"links"`
		} `json:"package"`
	} `json:"objects"`
}

// Search searches npm registry for packages via HTTP API
func (s *LibraryService) Search(query string) ([]model.LibraryInfo, error) {
	// Build registry URL with query parameters
	registryURL := "https://registry.npmjs.org/-/v1/search"
	params := url.Values{}
	params.Set("text", query)
	params.Set("size", "10")

	fullURL := registryURL + "?" + params.Encode()

	// Make HTTP GET request to npm registry
	resp, err := http.Get(fullURL)
	if err != nil {
		return []model.LibraryInfo{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []model.LibraryInfo{}, nil
	}

	// Read and parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	var result searchRegistry
	if err := json.Unmarshal(body, &result); err != nil {
		// Return empty result on parse error
		return []model.LibraryInfo{}, nil
	}

	limit := len(result.Objects)
	if limit > 10 {
		limit = 10
	}

	results := make([]model.LibraryInfo, 0, limit)
	for _, obj := range result.Objects[:limit] {
		results = append(results, model.LibraryInfo{
			Name:        obj.Package.Name,
			Version:     obj.Package.Version,
			Description: obj.Package.Description,
			Keywords:    obj.Package.Keywords,
			Homepage:    obj.Package.Links.Homepage,
			Repository:  obj.Package.Links.Repository,
			NPM:         obj.Package.Links.NPM,
			Downloads:   obj.Downloads.Weekly,
			Date:        obj.Package.Date,
		})
	}

	return results, nil
}

// Check checks if a package is available using pnpm view
func (s *LibraryService) Check(pkg string) (bool, error) {
	bin := "pnpm"
	if pm, ok := s.pm.(*PnpmPackageManager); ok {
		bin = pm.Bin
	}
	if err := ensureSupportedPnpm(bin); err != nil {
		return false, nil
	}

	cmd := exec.Command(bin, "view", pkg)
	cmd.Dir = s.dataDir
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		return false, nil // Package not found
	}

	return true, nil
}
