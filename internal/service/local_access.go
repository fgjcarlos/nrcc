package service

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

var hostnameSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

type localAccessRunner interface {
	LookPath(name string) (string, error)
	Run(dir string, name string, args ...string) (string, error)
}

type LocalAccessService struct {
	runner localAccessRunner
	port   int
	name   string
	tld    string

	mu     sync.RWMutex
	status model.LocalAccessStatus
	ready  bool
}

func NewLocalAccessService(port int) *LocalAccessService {
	return &LocalAccessService{
		runner: platform.NewRunner(),
		port:   port,
		name:   normalizeHostnameLabel(envOrFallback("NRCC_LOCAL_HOSTNAME", "nrcc"), "nrcc"),
		tld:    normalizeHostnameLabel(envOrFallback("NRCC_LOCAL_TLD", "localhost"), "localhost"),
	}
}

func (s *LocalAccessService) Detect() model.LocalAccessStatus {
	fallbackURL := fmt.Sprintf("http://127.0.0.1:%d", s.port)
	hostname := fmt.Sprintf("%s.%s", s.name, s.tld)
	status := model.LocalAccessStatus{
		Mode:        "direct",
		Hostname:    hostname,
		URL:         fallbackURL,
		FallbackURL: fallbackURL,
		Operational: true,
		Message:     fmt.Sprintf("Direct local access is available at %s", fallbackURL),
	}

	if _, err := s.runner.LookPath("portless"); err != nil {
		status.Message = fmt.Sprintf("portless is not installed; use %s or install it with `npm install -g portless`", fallbackURL)
		return status
	}

	status.Mode = "portless"
	status.URL = fmt.Sprintf("https://%s", hostname)
	status.PortlessAvailable = true
	status.Operational = false
	status.Message = fmt.Sprintf("portless is available. Start or restart NRCC to publish %s", status.URL)
	return status
}

func (s *LocalAccessService) EnsureConfigured() model.LocalAccessStatus {
	status := s.Detect()
	if !status.PortlessAvailable {
		s.store(status)
		return status
	}

	if _, err := s.runner.Run("", "portless", "proxy", "start"); err != nil {
		status.Mode = "direct"
		status.URL = status.FallbackURL
		status.Message = fmt.Sprintf("portless is installed but the proxy could not start (%s). Falling back to %s", err.Error(), status.FallbackURL)
		s.store(status)
		return status
	}

	if _, err := s.runner.Run("", "portless", "alias", s.name, strconv.Itoa(s.port), "--force"); err != nil {
		status.Mode = "direct"
		status.URL = status.FallbackURL
		status.Message = fmt.Sprintf("portless proxy is running but the route could not be registered (%s). Falling back to %s", err.Error(), status.FallbackURL)
		s.store(status)
		return status
	}

	status.Configured = true
	status.Operational = true
	status.Message = fmt.Sprintf("Stable local hostname configured at %s", status.URL)
	s.store(status)
	return status
}

func (s *LocalAccessService) Status() model.LocalAccessStatus {
	s.mu.RLock()
	if s.ready {
		defer s.mu.RUnlock()
		return s.status
	}
	s.mu.RUnlock()
	return s.Detect()
}

func (s *LocalAccessService) store(status model.LocalAccessStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.ready = true
}

func normalizeHostnameLabel(value string, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = hostnameSanitizer.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return fallback
	}
	return value
}

func envOrFallback(key, fallback string) string {
	if value := os.Getenv(key); strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
