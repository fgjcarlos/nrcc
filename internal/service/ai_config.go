package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fgjcarlos/nrcc/internal/store"
)

var (
	ErrAIEncryptionKeyRequired = errors.New("NRCC_ENCRYPTION_KEY is not configured; cannot store an AI provider API key")
	ErrAIConfigIncomplete      = errors.New("AI provider configuration is incomplete")
	ErrAIInvalidProvider       = errors.New("AI provider is not supported")
	ErrAIInvalidEndpoint       = errors.New("AI provider endpoint is invalid")
	ErrAIProbeTimeout          = errors.New("AI provider connection probe timed out")
	ErrAIProbeUnreachable      = errors.New("AI provider connection probe failed")
)

const defaultAIProbeTimeout = 5 * time.Second

type persistedAIConfig struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	Endpoint string `json:"endpoint,omitempty"`
	Model    string `json:"model,omitempty"`
	APIKey   string `json:"apiKey,omitempty"`
}

// AIConfigView is the safe configuration representation for API consumers.
type AIConfigView struct {
	Enabled          bool   `json:"enabled"`
	Provider         string `json:"provider"`
	Endpoint         string `json:"endpoint,omitempty"`
	Model            string `json:"model,omitempty"`
	APIKeyConfigured bool   `json:"apiKeyConfigured"`
}

// AIProviderProbe is the injected remote connectivity boundary.
type AIProviderProbe func(context.Context, AIConfig) error

// AIConfigServiceOption configures an AIConfigService.
type AIConfigServiceOption func(*AIConfigService)

func WithAIProviderProbe(probe AIProviderProbe) AIConfigServiceOption {
	return func(s *AIConfigService) {
		if probe != nil {
			s.probe = probe
		}
	}
}

func WithAIProbeTimeout(timeout time.Duration) AIConfigServiceOption {
	return func(s *AIConfigService) {
		if timeout > 0 {
			s.timeout = timeout
		}
	}
}

// AIConfigService persists separate, encrypted AI provider settings. It never
// reads or writes Node-RED environment configuration through EnvService.
type AIConfigService struct {
	store         *store.JSONStore[persistedAIConfig]
	encryptionKey string
	probe         AIProviderProbe
	timeout       time.Duration
}

func NewAIConfigService(dataDir, encryptionKey string, opts ...AIConfigServiceOption) *AIConfigService {
	s := &AIConfigService{
		store:         store.NewJSONStore[persistedAIConfig](filepath.Join(dataDir, "ai-config.json")),
		encryptionKey: encryptionKey,
		probe:         defaultAIProviderProbe,
		timeout:       defaultAIProbeTimeout,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Save replaces the persisted provider configuration. A supplied API key is
// always AES-GCM encrypted before JSONStore atomically writes its 0600 file.
func (s *AIConfigService) Save(cfg AIConfig) error {
	cfg = normalizeAIConfig(cfg)
	if !isSupportedAIProvider(cfg.Provider) {
		return fmt.Errorf("%w: %s", ErrAIInvalidProvider, cfg.Provider)
	}

	persisted := persistedAIConfig{Enabled: cfg.Enabled, Provider: cfg.Provider, Endpoint: cfg.Endpoint, Model: cfg.Model}
	if cfg.APIKey != "" {
		if s.encryptionKey == "" {
			return ErrAIEncryptionKeyRequired
		}
		ciphertext, err := Encrypt(cfg.APIKey, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("encrypt AI provider API key: %w", err)
		}
		persisted.APIKey = ciphertext
	}
	if err := s.store.Write(persisted); err != nil {
		return fmt.Errorf("persist AI provider configuration: %w", err)
	}
	return nil
}

// Load resolves persisted settings first. The legacy NRCC_AI_* bootstrap is
// used only when no dedicated persisted configuration file exists.
func (s *AIConfigService) Load() (AIConfig, error) {
	persisted, err := s.store.Read()
	if errors.Is(err, os.ErrNotExist) {
		cfg := normalizeAIConfig(LoadAIConfigFromEnv())
		if !isSupportedAIProvider(cfg.Provider) {
			return AIConfig{}, fmt.Errorf("%w: %s", ErrAIInvalidProvider, cfg.Provider)
		}
		return cfg, nil
	}
	if err != nil {
		return AIConfig{}, fmt.Errorf("read AI provider configuration: %w", err)
	}
	if !isSupportedAIProvider(persisted.Provider) {
		return AIConfig{}, fmt.Errorf("%w: %s", ErrAIInvalidProvider, persisted.Provider)
	}
	cfg := AIConfig{Enabled: persisted.Enabled, Provider: persisted.Provider, Endpoint: persisted.Endpoint, Model: persisted.Model}
	if persisted.APIKey == "" {
		return cfg, nil
	}
	if s.encryptionKey == "" {
		return AIConfig{}, ErrAIEncryptionKeyRequired
	}
	key, err := Decrypt(persisted.APIKey, s.encryptionKey)
	if err != nil {
		return AIConfig{}, fmt.Errorf("decrypt AI provider API key: %w", err)
	}
	cfg.APIKey = key
	return cfg, nil
}

func (s *AIConfigService) View() (AIConfigView, error) {
	cfg, err := s.Load()
	if err != nil {
		return AIConfigView{}, err
	}
	return AIConfigView{Enabled: cfg.Enabled, Provider: cfg.Provider, Endpoint: cfg.Endpoint, Model: cfg.Model, APIKeyConfigured: cfg.APIKey != ""}, nil
}

// Probe validates the active configuration then executes a bounded provider
// connection probe. Offline mode is intentionally local and needs no key.
func (s *AIConfigService) Probe(ctx context.Context) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return ErrAIDisabled
	}
	if cfg.Provider == "offline" {
		return nil
	}
	if cfg.Endpoint == "" || cfg.Model == "" {
		return ErrAIConfigIncomplete
	}
	if cfg.APIKey == "" {
		return ErrAIKeyRequired
	}
	if !validAIEndpoint(cfg.Endpoint) {
		return ErrAIInvalidEndpoint
	}
	probeCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	if err := s.probe(probeCtx, cfg); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w: %v", ErrAIProbeTimeout, err)
		}
		return fmt.Errorf("%w: %v", ErrAIProbeUnreachable, err)
	}
	return nil
}

func normalizeAIConfig(cfg AIConfig) AIConfig {
	cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	cfg.Model = strings.TrimSpace(cfg.Model)
	return cfg
}

func isSupportedAIProvider(provider string) bool {
	return provider == "offline" || provider == "openai"
}

func validAIEndpoint(endpoint string) bool {
	u, err := url.ParseRequestURI(endpoint)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

func defaultAIProviderProbe(ctx context.Context, cfg AIConfig) error {
	body, err := json.Marshal(map[string]interface{}{
		"model":      cfg.Model,
		"messages":   []AIMessage{{Role: "user", Content: "Connection test"}},
		"max_tokens": 1,
	})
	if err != nil {
		return fmt.Errorf("encode provider connection probe: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("provider returned %s", resp.Status)
	}
	return nil
}
