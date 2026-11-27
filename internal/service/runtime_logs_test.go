package service

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRuntimeLogBufferBoundsAndParses(t *testing.T) {
	b := newRuntimeLogBuffer(2, 128)
	b.add("12 Jan 12:34:56 - [error] boom")
	parsed := b.page("", 1, nil, "", time.Time{}, time.Time{})
	if len(parsed.Entries) != 1 || parsed.Entries[0].Level != "error" || parsed.Entries[0].Message != "boom" {
		t.Fatalf("parse failed: %#v", parsed.Entries)
	}
	b.add("12 Jan 12:34:57 - plain")
	b.add(strings.Repeat("x", 1024))
	page := b.page("", 10, nil, "", time.Time{}, time.Time{})
	if len(page.Entries) != 2 || page.Entries[0].Level != "unknown" || len(page.Entries[0].Message) > maxRuntimeLogLine {
		t.Fatalf("unexpected bounded newest entries: %#v", page.Entries)
	}
}

func TestRuntimeLogRingClearsEvictedSlot(t *testing.T) {
	b := newRuntimeLogBuffer(1, 1<<20)
	b.add("sensitive payload")
	b.add("new")
	if len(b.entries) != 1 || b.entries[0].Message != "new" {
		t.Fatalf("evicted payload remains referenced: %#v", b.entries)
	}
}

func TestRuntimeLogBufferFiltersAndEdgeBounds(t *testing.T) {
	t.Setenv("EDGE_MODE", "true")
	if runtimeLogCapacity() != 500 || runtimeLogBytes() != 1<<20 {
		t.Fatal("EDGE_MODE should use smaller bounded retention")
	}
	b := newRuntimeLogBuffer(4, 1<<20)
	b.add("line one")
	b.add("line two")
	page := b.page("", 4, []string{"unknown"}, "TWO", time.Time{}, time.Time{})
	if len(page.Entries) != 1 || page.Entries[0].Message != "line two" {
		t.Fatalf("filters failed: %#v", page.Entries)
	}
}

func TestRuntimeLogSparseFilterPaginationFindsOlderMatches(t *testing.T) {
	b := newRuntimeLogBuffer(8, 1<<20)
	b.add("needle-old")
	for i := 0; i < 5; i++ {
		b.add("other")
	}
	first := b.page("", 2, nil, "needle", time.Time{}, time.Time{})
	if len(first.Entries) != 1 || first.Entries[0].Message != "needle-old" {
		t.Fatalf("sparse page lost older match: %#v", first.Entries)
	}
}

func TestRuntimeLogsSinceRestartFiltersRetainedHistory(t *testing.T) {
	b := newRuntimeLogBuffer(8, 1<<20)
	b.add("old")
	b.reset()
	b.add("new")
	page := b.pageSince("", 8, nil, "", time.Time{}, time.Time{}, true)
	if len(page.Entries) != 1 || page.Entries[0].Message != "new" {
		t.Fatalf("since restart failed: %#v", page.Entries)
	}
}

func TestRuntimeLogsKeepHistoryAcrossRestart(t *testing.T) {
	b := newRuntimeLogBuffer(8, 1<<20)
	b.add("before restart")
	first := b.page("", 8, nil, "", time.Time{}, time.Time{})
	b.reset()
	b.add("after restart")
	page := b.page("", 8, nil, "", time.Time{}, time.Time{})
	if first.Generation != page.Generation || len(page.Entries) != 2 {
		t.Fatalf("restart discarded/changed history: before=%#v after=%#v", first, page)
	}
	if page.StartedAt.IsZero() {
		t.Fatal("missing last startedAt")
	}
}

func TestParseRuntimeLogConventionalTimestampAndLevels(t *testing.T) {
	for _, tc := range []struct {
		level string
		want  string
	}{{"trace", "trace"}, {"fatal", "fatal"}, {"info", "info"}, {"wat", "unknown"}} {
		ts, level, msg := parseRuntimeLog("27 Nov 14:03:05 - [" + tc.level + "] message")
		wantMessage := "message"
		if tc.want == "unknown" {
			wantMessage = "27 Nov 14:03:05 - [wat] message"
		}
		if level != tc.want || msg != wantMessage {
			t.Errorf("level %s: got %s %s", tc.level, level, msg)
		}
		if tc.level == "info" && (ts.Year() != time.Now().Year() || ts.Month() != time.November || ts.Day() != 27 || ts.Hour() != 14 || ts.Minute() != 3 || ts.Second() != 5) {
			t.Errorf("timestamp not parsed: %s", ts)
		}
	}
}

func TestRuntimeLogCursorUsesGenerationAndSequence(t *testing.T) {
	b := newRuntimeLogBuffer(20, 1<<20)
	for i := 0; i < 12; i++ {
		b.add("entry")
	}
	first := b.page("", 10, nil, "", time.Time{}, time.Time{})
	if first.Entries[0].Message != "entry" {
		t.Fatal("missing entries")
	}
	cursor := first.NextCursor
	next := b.page(cursor, 10, nil, "", time.Time{}, time.Time{})
	if len(next.Entries) != 2 || next.Entries[0].ID != strings.Split(cursor, ":")[0]+":1" || next.Entries[1].ID != strings.Split(cursor, ":")[0]+":2" {
		t.Fatalf("cursor order incorrect: %#v", next.Entries)
	}
}

func TestForwardOutputCapturePreservesOverlongForwarding(t *testing.T) {
	line := strings.Repeat("x", maxRuntimeLogLine+200) + "\n"
	var dst bytes.Buffer
	logs := newRuntimeLogBuffer(2, 1<<20)
	forwardOutputCapture(&dst, strings.NewReader(line), "node-red: ", logs, "stderr")
	if dst.String() != "node-red: "+line {
		t.Fatal("capture altered container log forwarding")
	}
	entry := logs.page("", 1, nil, "", time.Time{}, time.Time{}).Entries[0]
	if len(entry.Message) != maxRuntimeLogLine || entry.Source != "stderr" {
		t.Fatalf("unexpected captured line: length=%d source=%s", len(entry.Message), entry.Source)
	}
}
