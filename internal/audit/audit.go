package audit

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	fileName   = "audit.jsonl"
	maxSize    = 10 * 1024 * 1024 // 10 MB
	maxBackups = 5
)

type Event struct {
	Timestamp string            `json:"ts"`
	Actor     string            `json:"actor"`
	Action    string            `json:"action"`
	Target    string            `json:"target,omitempty"`
	IP        string            `json:"ip"`
	UserAgent string            `json:"ua"`
	Result    string            `json:"result"`
	Meta      map[string]string `json:"meta,omitempty"`
}

type Service struct {
	mu   sync.Mutex
	dir  string
	file *os.File
	size int64
}

func NewService(dataDir string) (*Service, error) {
	dir := filepath.Join(dataDir, "audit")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create audit dir: %w", err)
	}

	s := &Service{dir: dir}
	if err := s.openLog(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Log(r *http.Request, actor, action, target, result string, meta map[string]string) {
	if s == nil {
		return
	}

	event := Event{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Actor:     actor,
		Action:    action,
		Target:    target,
		IP:        extractIP(r),
		UserAgent: r.Header.Get("User-Agent"),
		Result:    result,
		Meta:      meta,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return
	}

	n, err := s.file.Write(data)
	if err != nil {
		return
	}
	s.size += int64(n)

	if s.size >= maxSize {
		s.rotate()
	}
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func (s *Service) openLog() error {
	path := filepath.Join(s.dir, fileName)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open audit log: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	s.file = f
	s.size = info.Size()
	return nil
}

func (s *Service) rotate() {
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}

	current := filepath.Join(s.dir, fileName)
	rotated := filepath.Join(s.dir, fmt.Sprintf("audit-%s.jsonl", time.Now().UTC().Format("20060102-150405")))
	if err := os.Rename(current, rotated); err != nil {
		// best-effort rotation; audit appends reopen on next write
		_ = err
	}

	s.pruneOld()

	s.size = 0
	if err := s.openLog(); err != nil {
		// rotation failure surfaces on next append; do not crash
		_ = err
	}
}

func (s *Service) pruneOld() {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}

	var rotated []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "audit-") && strings.HasSuffix(name, ".jsonl") {
			rotated = append(rotated, name)
		}
	}

	if len(rotated) <= maxBackups {
		return
	}

	sort.Strings(rotated)
	for _, name := range rotated[:len(rotated)-maxBackups] {
		_ = os.Remove(filepath.Join(s.dir, name))
	}
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
