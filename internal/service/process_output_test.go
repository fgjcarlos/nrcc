//go:build linux

package service

import (
	"bytes"
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer safe for the two forwarding goroutines.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Issue #1015: Node-RED's stdout and stderr reach the NRCC process output
// (the container log) with a "node-red: " prefix on every line.
func TestStartForwardsNodeRedOutput(t *testing.T) {
	pm, script := crashingNodeRed(t)
	body := "#!/bin/sh\necho 'Welcome to Node-RED'\necho 'SyntaxError: Unexpected token' >&2\nprintf 'no newline'\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil { //nolint:gosec // G306 the stub must be executable.
		t.Fatal(err)
	}
	pm.maxRestarts = 0
	out := &syncBuffer{}
	pm.output = out

	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	want := []string{
		"node-red: Welcome to Node-RED\n",
		"node-red: SyntaxError: Unexpected token\n",
		"node-red: no newline\n",
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		got := out.String()
		ok := true
		for _, w := range want {
			ok = ok && strings.Contains(got, w)
		}
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	_ = pm.Stop(context.Background())
	got := out.String()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("forwarded output missing %q; got %q", w, got)
		}
	}
}

// slowWriter delays each Write so the child exits while output is unread.
type slowWriter struct{ syncBuffer }

func (w *slowWriter) Write(p []byte) (int, error) {
	time.Sleep(time.Millisecond)
	return w.syncBuffer.Write(p)
}

// cmd.Wait must not close the output pipes while the forwarders still read
// them: a child that exits while its output is still unread must still get
// its last line into the log (#1015 follow-up).
func TestStartForwardsOutputTailAfterExit(t *testing.T) {
	pm, script := crashingNodeRed(t)
	body := "#!/bin/sh\ni=0\nwhile [ $i -lt 500 ]; do echo \"line $i padding padding padding\"; i=$((i+1)); done\necho 'LAST LINE'\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil { //nolint:gosec // G306 the stub must be executable.
		t.Fatal(err)
	}
	pm.maxRestarts = 0
	out := &slowWriter{}
	pm.output = out

	if err := pm.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(out.String(), "node-red: LAST LINE\n") {
		time.Sleep(10 * time.Millisecond)
	}
	_ = pm.Stop(context.Background())
	if got := out.String(); !strings.Contains(got, "node-red: LAST LINE\n") {
		t.Fatalf("last line lost; forwarded %d bytes, tail %q", len(got), got[max(0, len(got)-120):])
	}
}

func TestForwardOutputPrefixesEachLine(t *testing.T) {
	long := strings.Repeat("x", 10000)
	src := strings.NewReader("a\n\nb\n" + long + "\ntail")
	var dst bytes.Buffer
	forwardOutput(&dst, src, "node-red: ")
	want := "node-red: a\nnode-red: \nnode-red: b\nnode-red: " + long + "\nnode-red: tail\n"
	if dst.String() != want {
		t.Fatalf("forwardOutput mismatch:\n got %q\nwant %q", dst.String(), want)
	}
}
