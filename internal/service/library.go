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

	"github.com/fgjcarlos/nrcc/internal/model"
)

// LibraryService handles package library operations with npm.
type LibraryService struct {
	dataDir string
	pm      PackageManager
	// restart is an optional Node-RED restart hook fired after Install /
	// Uninstall so the new set of nodes is picked up by the running
	// process. nil disables the restart (external-Node-RED setups or
	// tests). ponytail: full Stop+Start on every install — fine while
	// installs are operator-initiated and infrequent; switch to a
	// debounced signal if installs become background-driven.
	restart func() error
}

// NewLibraryService creates a new library service with default npm package manager
func NewLibraryService(dataDir string) *LibraryService {
	return NewLibraryServiceWithPackageManager(dataDir, NewNpmPackageManager(dataDir))
}

// NewLibraryServiceWithPackageManager creates a new library service with a custom package manager
func NewLibraryServiceWithPackageManager(dataDir string, pm PackageManager) *LibraryService {
	return &LibraryService{
		dataDir: dataDir,
		pm:      pm,
	}
}

// SetNodeRedRestart wires an optional Node-RED restart hook. Call this once
// during startup with the ProcessManager's restart closure so Install /
// Uninstall can pick up new nodes without restarting the container.
func (s *LibraryService) SetNodeRedRestart(restart func() error) {
	s.restart = restart
}

// Install installs a package using npm install and (best-effort) restarts
// Node-RED so the running editor picks up the new node.
func (s *LibraryService) Install(pkg string) error {
	if err := s.pm.Install(pkg); err != nil {
		return err
	}
	s.fireRestart()
	return nil
}

// Uninstall uninstalls a package using npm uninstall and (best-effort)
// restarts Node-RED.
func (s *LibraryService) Uninstall(pkg string) error {
	if err := s.pm.Uninstall(pkg); err != nil {
		return err
	}
	s.fireRestart()
	return nil
}

// fireRestart invokes the configured restart hook, ignoring the error.
// Install/Uninstall must not fail when the runtime is unavailable
// (external Node-RED, tests, dev loop with no running NR); the operator
// can always restart manually.
func (s *LibraryService) fireRestart() {
	if s.restart == nil {
		return
	}
	_ = s.restart()
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

			// Enrich with metadata from node_modules; ignore errors because
					// metadata is optional and the caller already has the package
					// name and version from the directory listing.
					_ = s.enrichLibraryMetadata(&lib)

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
	if name, ok := pkg.Author.(string); ok {
		lib.Author = name
	} else if obj, ok := pkg.Author.(map[string]interface{}); ok {
		if n, ok := obj["name"].(string); ok {
			lib.Author = n
		}
	}
	lib.License = pkg.License

	// Handle repository field — can be string or object with "url" property
	if url, ok := pkg.Repository.(string); ok {
		lib.Repository = url
	} else if obj, ok := pkg.Repository.(map[string]interface{}); ok {
		if u, ok := obj["url"].(string); ok {
			lib.Repository = u
		}
	}

	return nil
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
	defer func() { _ = resp.Body.Close() }()

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

// Check reports whether a package is available in the npm registry using
// `npm view`. It returns (false, err) only when npm itself cannot run. A
// non-zero `npm view` exit — whether the package does not exist OR the
// registry is unreachable — is reported as (false, nil); these two cases
// are not currently distinguished.
func (s *LibraryService) Check(pkg string) (bool, error) {
	// Validate before invoking npm: `npm view` would otherwise resolve local
	// paths (file:...), URLs and git refs from the raw URL parameter.
	if err := ValidatePackageName(pkg); err != nil {
		return false, err
	}

	bin := "npm"
	if pm, ok := s.pm.(*NpmPackageManager); ok {
		bin = pm.Bin
	}
	if err := ensureNpm(bin); err != nil {
		return false, err
	}

	cmd := exec.Command(bin, "view", pkg)
	cmd.Dir = s.dataDir
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		return false, nil // Package not found in registry
	}

	return true, nil
}
