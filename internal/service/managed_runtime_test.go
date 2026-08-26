package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveManagedRuntime_ImageLocal(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "usr", "src", "node-red", "node_modules", ".bin", "node-red")
	if err := os.MkdirAll(filepath.Dir(executable), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	managed := ResolveManagedRuntime(executable, root)
	if managed.Strategy != StrategyImageLocal {
		t.Fatalf("expected image-local strategy, got %q", managed.Strategy)
	}
	if managed.Executable != executable {
		t.Fatalf("expected executable %q, got %q", executable, managed.Executable)
	}
	wantRoot := filepath.Join(root, "usr", "src", "node-red", "node_modules", "node-red")
	if managed.TreeRoot != wantRoot {
		t.Fatalf("expected tree root %q, got %q", wantRoot, managed.TreeRoot)
	}
}

func TestResolveManagedRuntime_NpmGlobal(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "usr", "local", "lib", "node_modules", "node-red", "bin", "node-red.js")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("#!/usr/bin/env node\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(root, "usr", "local", "bin", "node-red")
	if err := os.MkdirAll(filepath.Dir(executable), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, executable); err != nil {
		t.Fatal(err)
	}

	managed := ResolveManagedRuntime(executable, root)
	if managed.Strategy != StrategyNpmGlobal {
		t.Fatalf("expected npm-global strategy, got %q", managed.Strategy)
	}
	if managed.Executable != executable {
		t.Fatalf("expected executable %q, got %q", executable, managed.Executable)
	}
	wantRoot := filepath.Join(root, "usr", "local", "lib", "node_modules", "node-red")
	if managed.TreeRoot != wantRoot {
		t.Fatalf("expected tree root %q, got %q", wantRoot, managed.TreeRoot)
	}
}

func TestResolveManagedRuntime_External(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{name: "empty command", cmd: ""},
		{name: "unknown path", cmd: filepath.Join(t.TempDir(), "opt", "node-red")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			managed := ResolveManagedRuntime(tt.cmd, t.TempDir())
			if managed.Strategy != StrategyExternal {
				t.Fatalf("expected external strategy, got %q", managed.Strategy)
			}
		})
	}
}
