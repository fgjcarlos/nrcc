package platform

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Runner struct {
	Timeout time.Duration
}

func NewRunner() Runner {
	return Runner{Timeout: 20 * time.Second}
}

func (r Runner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (r Runner) Run(dir string, name string, args ...string) (string, error) {
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(stderr.String())
		if output == "" {
			output = strings.TrimSpace(stdout.String())
		}
		if output != "" {
			return "", fmt.Errorf("%w: %s", err, output)
		}
		return "", err
	}

	if stderr.Len() > 0 {
		return strings.TrimSpace(stderr.String()), nil
	}

	return strings.TrimSpace(stdout.String()), nil
}
