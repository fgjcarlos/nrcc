package audit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fgjcarlos/nrcc/internal/middleware"
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

type auditFile interface {
	io.Writer
	Close() error
	Stat() (os.FileInfo, error)
}

type fileSystem interface {
	MkdirAll(string, os.FileMode) error
	OpenFile(string, int, os.FileMode) (auditFile, error)
	Link(string, string) error
	Remove(string) error
	ReadDir(string) ([]os.DirEntry, error)
	Stat(string) (os.FileInfo, error)
}

type osFileSystem struct{}

func (*osFileSystem) MkdirAll(path string, perm os.FileMode) error { return os.MkdirAll(path, perm) }
func (*osFileSystem) OpenFile(path string, flag int, perm os.FileMode) (auditFile, error) {
	// #nosec G304 -- path is built from operator-supplied audit directory and controlled filenames.
	return os.OpenFile(path, flag, perm)
}
func (*osFileSystem) Link(oldname, newname string) error { return os.Link(oldname, newname) }
func (*osFileSystem) Remove(path string) error           { return os.Remove(path) }
func (*osFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}
func (*osFileSystem) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }

type Reporter interface{ Report(Report) }

type ReporterFunc func(Report)

func (f ReporterFunc) Report(report Report) { f(report) }

type Report struct {
	Persisted bool
	Kind      string
	Stage     string
	Err       error
	Secondary []SecondaryError
}

type Outcome struct {
	Persisted      bool
	EventErr       *OpError
	MaintenanceErr *OpError
}

type SecondaryError struct {
	Stage string
	Err   error
}

type OpError struct {
	Stage     string
	Primary   error
	Secondary []SecondaryError
}

func (e *OpError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("audit %s: %v", e.Stage, e.Primary)
}

func (e *OpError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Primary
}

type dependencies struct {
	now      func() time.Time
	fs       fileSystem
	reporter Reporter
}

type Service struct {
	mu            sync.Mutex
	dir           string
	file          auditFile
	size          int64
	deps          dependencies
	pendingBackup string
}

func NewService(dataDir string) (*Service, error) {
	return newService(dataDir, dependencies{
		now: time.Now,
		fs:  &osFileSystem{},
		reporter: ReporterFunc(func(report Report) {
			log.Printf("audit degradation: kind=%s persisted=%t stage=%s err=%v secondary=%v", report.Kind, report.Persisted, report.Stage, report.Err, report.Secondary)
		}),
	})
}

func newService(dataDir string, deps dependencies) (*Service, error) {
	dir := filepath.Join(dataDir, "audit")
	if err := deps.fs.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create audit dir: %w", err)
	}

	s := &Service{dir: dir, deps: deps}
	if err := s.openActive(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Log(r *http.Request, actor, action, target, result string, meta map[string]string) Outcome {
	if s == nil {
		return eventFailure("unavailable-writer", errors.New("audit service is unavailable"))
	}

	// IP is resolved through middleware.ExtractIP, which honors
	// NRCC_TRUSTED_PROXIES. Untrusted peers cannot spoof XFF.
	ip := middleware.ExtractIP(r)

	event := Event{
		Timestamp: s.deps.now().UTC().Format(time.RFC3339),
		Actor:     actor,
		Action:    action,
		Target:    target,
		IP:        ip,
		UserAgent: r.Header.Get("User-Agent"),
		Result:    result,
		Meta:      meta,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return s.failed(eventFailure("marshal", err))
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		if err := s.openActive(); err != nil {
			return s.failed(eventFailure("unavailable-writer", err))
		}
	}

	n, err := s.file.Write(data)
	if err != nil {
		return s.failed(eventFailure("write", err))
	}
	if n != len(data) {
		return s.failed(eventFailure("write", io.ErrShortWrite))
	}
	s.size += int64(n)

	if s.size >= maxSize {
		if maintenanceErr := s.rotate(); maintenanceErr != nil {
			return s.failed(Outcome{Persisted: true, MaintenanceErr: maintenanceErr})
		}
	}
	return Outcome{Persisted: true}
}

func eventFailure(stage string, err error) Outcome {
	return Outcome{EventErr: &OpError{Stage: stage, Primary: err}}
}

func (s *Service) failed(outcome Outcome) Outcome {
	opErr := outcome.EventErr
	kind := "event"
	if opErr == nil {
		opErr = outcome.MaintenanceErr
		kind = "maintenance"
	}
	if s.deps.reporter != nil {
		s.deps.reporter.Report(Report{Persisted: outcome.Persisted, Kind: kind, Stage: opErr.Stage, Err: opErr.Primary, Secondary: opErr.Secondary})
	}
	return outcome
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		s.size = 0
		return err
	}
	return nil
}

func (s *Service) openActive() error {
	// #nosec G304 -- path is built from operator-supplied audit directory + a constant filename; not request-derived.
	path := filepath.Join(s.dir, fileName)
	// #nosec G304 -- see path derivation above.
	f, err := s.deps.fs.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
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

func (s *Service) selectBackupName() (string, error) {
	prefix := s.deps.now().UTC().Format("20060102-150405.000000000")
	for ordinal := 0; ordinal <= 999999; ordinal++ {
		candidate := filepath.Join(s.dir, fmt.Sprintf("audit-%s-%06d.jsonl", prefix, ordinal))
		if _, err := s.deps.fs.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("inspect backup candidate: %w", err)
		}
	}
	return "", errors.New("audit backup ordinal exhausted")
}

func (s *Service) rotate() *OpError {
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			s.file = nil
			s.size = 0
			return &OpError{Stage: "close-active", Primary: err}
		}
		s.file = nil
		s.size = 0
	}
	candidate, err := s.selectBackupName()
	if err != nil {
		_ = s.openActive()
		return &OpError{Stage: "select-backup", Primary: err}
	}
	active := filepath.Join(s.dir, fileName)
	if err := s.deps.fs.Link(active, candidate); err != nil {
		secondary := s.recoverActive()
		return &OpError{Stage: "link-backup", Primary: err, Secondary: secondary}
	}
	if err := s.deps.fs.Remove(active); err != nil {
		rollbackErr := s.deps.fs.Remove(candidate)
		secondary := make([]SecondaryError, 0, 2)
		if rollbackErr != nil {
			s.pendingBackup = candidate
			secondary = append(secondary, SecondaryError{Stage: "rollback-backup", Err: rollbackErr})
		} else {
			secondary = append(secondary, s.recoverActive()...)
		}
		return &OpError{Stage: "remove-active", Primary: err, Secondary: secondary}
	}
	if err := s.openActive(); err != nil {
		return &OpError{Stage: "open-replacement", Primary: err}
	}
	return nil
}

func (s *Service) recoverActive() []SecondaryError {
	if err := s.openActive(); err != nil {
		return []SecondaryError{{Stage: "recover-active", Err: err}}
	}
	return nil
}
