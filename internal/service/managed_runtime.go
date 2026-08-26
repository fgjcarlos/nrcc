package service

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// RuntimeStrategy identifies how NRCC's managed Node-RED is installed.
type RuntimeStrategy string

const (
	StrategyImageLocal RuntimeStrategy = "image-local"
	StrategyNpmGlobal  RuntimeStrategy = "npm-global"
	StrategyExternal   RuntimeStrategy = "external"
)

var (
	ErrVersionUndetectable       = errors.New("managed Node-RED version undetectable")
	ErrUpdateUnsupportedForImage = errors.New("in-place update unsupported: managed Node-RED is embedded in the container image; rebuild the image with a newer Node-RED tag instead")
	ErrExternalRuntimeUnmanaged  = errors.New("update unsupported: Node-RED is externally managed")
	ErrUpdateVerificationFailed  = errors.New("update verification failed: managed Node-RED version did not change to the requested target")
)

// ManagedRuntime describes the Node-RED executable controlled by NRCC.
type ManagedRuntime struct {
	Executable string
	TreeRoot   string
	Strategy   RuntimeStrategy
}

// ResolveManagedRuntime derives the installation strategy from the executable
// NRCC will launch. dataDir is reserved for layouts rooted in managed data.
func ResolveManagedRuntime(nodeRedCmd, dataDir string) ManagedRuntime {
	_ = dataDir
	executable := resolveExecutable(nodeRedCmd)
	managed := ManagedRuntime{Executable: executable, Strategy: StrategyExternal}
	if executable == "" {
		return managed
	}

	resolved := executable
	if target, err := filepath.EvalSymlinks(executable); err == nil {
		resolved = target
	}

	if root, ok := pathThroughMarker(executable, filepath.Join("usr", "src", "node-red")); ok {
		managed.TreeRoot = filepath.Join(root, "node_modules", "node-red")
		managed.Strategy = StrategyImageLocal
		return managed
	}
	if root, ok := pathThroughMarker(resolved, filepath.Join("usr", "src", "node-red")); ok {
		managed.TreeRoot = filepath.Join(root, "node_modules", "node-red")
		managed.Strategy = StrategyImageLocal
		return managed
	}

	if root, ok := nodeRedPackageRoot(resolved); ok {
		managed.TreeRoot = root
		managed.Strategy = StrategyNpmGlobal
	}
	return managed
}

func resolveExecutable(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	if !filepath.IsAbs(command) {
		if found, err := exec.LookPath(command); err == nil {
			command = found
		}
	}
	abs, err := filepath.Abs(command)
	if err != nil {
		return filepath.Clean(command)
	}
	return filepath.Clean(abs)
}

func pathThroughMarker(path, marker string) (string, bool) {
	marker = string(filepath.Separator) + marker
	index := strings.Index(filepath.Clean(path), marker)
	if index < 0 {
		return "", false
	}
	end := index + len(marker)
	return filepath.Clean(path)[:end], true
}

func nodeRedPackageRoot(path string) (string, bool) {
	clean := filepath.Clean(path)
	marker := filepath.Join("lib", "node_modules", "node-red")
	root, ok := pathThroughMarker(clean, marker)
	if !ok {
		return "", false
	}
	return root, true
}
