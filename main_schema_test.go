package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStartupRejectsFutureSchema(t *testing.T) {
	if os.Getenv("NRCC_TEST_SCHEMA_STARTUP") == "1" {
		log.SetFlags(0)
		main()
		return
	}
	for _, name := range []string{"config.json", "cc-users.json", "backup_config.json"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, name)
			original := []byte(`{"schemaVersion":2,"futureField":"keep me"}`)
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			// #nosec G204 G702 -- re-executes this test binary with a fixed test selector.
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStartupRejectsFutureSchema$")
			cmd.Env = append(os.Environ(), "NRCC_TEST_SCHEMA_STARTUP=1", "DATA_DIR="+dir,
				"JWT_SECRET=", "PORT=0", "NRCC_MANAGE_NODE_RED=false", "NRCC_APPLY_READINESS_TIMEOUT=")
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 {
				t.Errorf("startup exit = %v; want status 1", err)
			}
			if stdout.Len() != 0 {
				t.Errorf("startup stdout contained %d bytes; want none", stdout.Len())
			}
			want := "Cannot start: " + name + " has schemaVersion 2, but this binary supports up to 1; start with a newer NRCC version\n"
			if stderr.String() != want {
				t.Errorf("startup stderr did not match %q", want)
			}
			// #nosec G304 -- path is a fixed fixture inside t.TempDir().
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, original) {
				t.Errorf("file changed: %q, %v", got, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Errorf("startup modified the data directory: %v, %v", entries, err)
			}
		})
	}
}
