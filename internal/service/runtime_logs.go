package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRuntimeLogLine = 4096

func edgeRuntimeLogs() bool { enabled, _ := strconv.ParseBool(os.Getenv("EDGE_MODE")); return enabled }
func runtimeLogCapacity() int {
	if edgeRuntimeLogs() {
		return 500
	}
	return 5000
}
func runtimeLogBytes() int {
	if edgeRuntimeLogs() {
		return 1 << 20
	}
	return 8 << 20
}

type RuntimeLogEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}
type RuntimeLogPage struct {
	Entries       []RuntimeLogEntry `json:"entries"`
	Generation    string            `json:"generation"`
	HighWatermark string            `json:"highWatermark"`
	NextCursor    string            `json:"nextCursor,omitempty"`
	StartedAt     time.Time         `json:"startedAt,omitempty"`
}
type runtimeLogBuffer struct {
	mu                                      sync.Mutex
	entries                                 []RuntimeLogEntry
	start, count, capacity, maxBytes, bytes int
	seq                                     uint64
	generation                              string
	startedAt                               time.Time
}

func newRuntimeLogBuffer(capacity, maxBytes int) *runtimeLogBuffer {
	if capacity < 1 {
		capacity = 1
	}
	if maxBytes < maxRuntimeLogLine {
		maxBytes = maxRuntimeLogLine
	}
	return &runtimeLogBuffer{entries: make([]RuntimeLogEntry, capacity), capacity: capacity, maxBytes: maxBytes, generation: newLogGeneration()}
}
func newLogGeneration() string {
	var id [16]byte
	_, _ = rand.Read(id[:])
	return hex.EncodeToString(id[:])
}
func (b *runtimeLogBuffer) reset()          { b.mu.Lock(); defer b.mu.Unlock(); b.startedAt = time.Now().UTC() }
func (b *runtimeLogBuffer) add(line string) { b.addSource(line, "stdout") }
func (b *runtimeLogBuffer) addSource(line, source string) {
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if len(line) > maxRuntimeLogLine {
		line = line[:maxRuntimeLogLine]
	}
	ts, level, msg := parseRuntimeLog(line)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	entry := RuntimeLogEntry{ID: fmt.Sprintf("%s:%d", b.generation, b.seq), Timestamp: ts, Level: level, Source: source, Message: msg}
	for b.count > 0 && (b.count == b.capacity || b.bytes+entryBytes(entry) > b.maxBytes) {
		b.bytes -= entryBytes(b.entries[b.start])
		b.entries[b.start] = RuntimeLogEntry{}
		b.start = (b.start + 1) % b.capacity
		b.count--
	}
	if entryBytes(entry) > b.maxBytes {
		return
	}
	idx := (b.start + b.count) % b.capacity
	b.entries[idx] = entry
	b.count++
	b.bytes += entryBytes(entry)
}
func entryBytes(e RuntimeLogEntry) int {
	return len(e.ID) + len(e.Level) + len(e.Message) + len(e.Source) + 128
}
func parseRuntimeLog(line string) (time.Time, string, string) {
	if i := strings.Index(line, " - ["); i >= 0 {
		stampText := strings.TrimSpace(line[:i])
		stamp, err := time.Parse(time.RFC3339, stampText)
		if err != nil {
			stamp, err = time.Parse("2 Jan 15:04:05", stampText)
			if err == nil {
				stamp = stamp.AddDate(time.Now().Year(), 0, 0)
			}
		}
		rest := line[i+4:]
		if end := strings.Index(rest, "] "); end >= 0 {
			level := strings.ToLower(rest[:end])
			switch level {
			case "info", "warn", "error", "debug", "trace", "fatal":
				if err != nil {
					stamp = time.Now().UTC()
				}
				return stamp.UTC(), level, rest[end+2:]
			}
		}
	}
	return time.Now().UTC(), "unknown", line
}
func parseLogSequence(id, generation string) (uint64, bool) {
	p := strings.Split(id, ":")
	if len(p) != 2 || p[0] != generation {
		return 0, false
	}
	n, e := strconv.ParseUint(p[1], 10, 64)
	return n, e == nil
}
func (b *runtimeLogBuffer) ValidCursor(cursor string) bool { return b.validCursor(cursor) }
func (b *runtimeLogBuffer) validCursor(cursor string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, ok := parseLogSequence(cursor, b.generation)
	return ok && n <= b.seq
}
func (b *runtimeLogBuffer) page(before string, limit int, levels []string, text string, from, to time.Time) RuntimeLogPage {
	return b.pageSince(before, limit, levels, text, from, to, false)
}
func (b *runtimeLogBuffer) pageSince(before string, limit int, levels []string, text string, from, to time.Time, sinceRestart bool) RuntimeLogPage {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	now := time.Now().UTC()
	matches := func(e RuntimeLogEntry) bool {
		if len(levels) > 0 {
			ok := false
			for _, l := range levels {
				if strings.EqualFold(l, e.Level) {
					ok = true
				}
			}
			if !ok {
				return false
			}
		}
		if text != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(text)) {
			return false
		}
		if sinceRestart && !b.startedAt.IsZero() && e.Timestamp.Before(b.startedAt) {
			return false
		}
		return (from.IsZero() || !e.Timestamp.Before(from)) && (to.IsZero() || !e.Timestamp.After(to))
	}
	var bound uint64 = b.seq
	if before != "" {
		if n, ok := parseLogSequence(before, b.generation); ok {
			bound = n - 1
		}
	}
	entries := make([]RuntimeLogEntry, 0, limit)
	var olderMatch bool
	for off := b.count - 1; off >= 0; off-- {
		e := b.entries[(b.start+off)%b.capacity]
		seq, _ := parseLogSequence(e.ID, b.generation)
		if seq > bound || !matches(e) {
			continue
		}
		if len(entries) < limit {
			entries = append(entries, e)
		} else {
			olderMatch = true
			break
		}
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	high := fmt.Sprintf("%s:%d", b.generation, b.seq)
	next := ""
	if olderMatch && len(entries) > 0 {
		next = entries[0].ID
	}
	started := b.startedAt
	if started.IsZero() {
		started = now
	}
	return RuntimeLogPage{Entries: entries, Generation: b.generation, HighWatermark: high, NextCursor: next, StartedAt: started}
}
func (b *runtimeLogBuffer) after(cursor string) []RuntimeLogEntry {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, ok := parseLogSequence(cursor, b.generation)
	if !ok {
		return nil
	}
	out := make([]RuntimeLogEntry, 0)
	for off := 0; off < b.count; off++ {
		e := b.entries[(b.start+off)%b.capacity]
		seq, _ := parseLogSequence(e.ID, b.generation)
		if seq > n {
			out = append(out, e)
			if len(out) == 100 {
				break
			}
		}
	}
	return out
}
func (pm *ProcessManager) RuntimeLogs(cursor string, limit int, levels []string, text string, from, to time.Time) RuntimeLogPage {
	return pm.logs.page(cursor, limit, levels, text, from, to)
}
func (pm *ProcessManager) RuntimeLogsSince(cursor string, limit int, levels []string, text string, from, to time.Time, sinceRestart bool) RuntimeLogPage {
	return pm.logs.pageSince(cursor, limit, levels, text, from, to, sinceRestart)
}
func (pm *ProcessManager) RuntimeLogsAfter(cursor string) []RuntimeLogEntry {
	return pm.logs.after(cursor)
}
func (pm *ProcessManager) RuntimeLogBuffer() *runtimeLogBuffer { return pm.logs }
func (b *runtimeLogBuffer) AddForTest(line string)             { b.add(line) }
