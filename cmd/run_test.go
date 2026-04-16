package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunHelpDoesNotListStop(t *testing.T) {
	output := captureStdout(t, func() {
		if err := Run([]string{"help"}, nil); err != nil {
			t.Fatalf("Run(help) error = %v", err)
		}
	})

	if strings.Contains(output, "  stop      ") {
		t.Fatalf("help output unexpectedly lists stop command:\n%s", output)
	}
	if !strings.Contains(output, "  start     Start the local control center") {
		t.Fatalf("help output missing start command:\n%s", output)
	}
}

func TestRunStopReturnsUnknownCommand(t *testing.T) {
	output := captureStdout(t, func() {
		err := Run([]string{"stop"}, nil)
		if err == nil {
			t.Fatal("Run(stop) error = nil, want unknown command")
		}
		if err.Error() != "unknown command: stop" {
			t.Fatalf("Run(stop) error = %q, want %q", err.Error(), "unknown command: stop")
		}
	})

	if strings.Contains(output, "  stop      ") {
		t.Fatalf("help output unexpectedly lists stop command:\n%s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close: %v", err)
	}

	return buf.String()
}
