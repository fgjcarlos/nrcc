package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func saveAIConfig(t *testing.T, svc *AIConfigService, cfg AIConfig) {
	t.Helper()
	if err := svc.Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
}

func loadAIConfig(t *testing.T, svc *AIConfigService) AIConfig {
	t.Helper()
	cfg, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	return cfg
}

func newConfiguredAIService(t *testing.T, opts ...AIConfigServiceOption) *AIConfigService {
	t.Helper()
	svc := NewAIConfigService(t.TempDir(), "encryption-key", opts...)
	saveAIConfig(t, svc, AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "key"})
	return svc
}

func TestAIConfigServicePersistsEncryptedSecretAndReturnsRedactedView(t *testing.T) {
	dir := t.TempDir()
	svc := NewAIConfigService(dir, "encryption-key")
	key := "test-api-key-must-not-be-persisted-raw"

	saveAIConfig(t, svc, AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test/v1", Model: "test-model", APIKey: key})

	view, err := svc.View()
	if err != nil {
		t.Fatalf("View() error: %v", err)
	}
	if !view.Enabled || view.Provider != "openai" || !view.APIKeyConfigured {
		t.Fatalf("unexpected redacted view: %#v", view)
	}
	encodedView, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal public view: %v", err)
	}
	if strings.Contains(string(encodedView), key) {
		t.Fatalf("public view leaked API key: %#v", view)
	}

	// #nosec G304 -- the path is derived from t.TempDir() and a literal filename.
	persisted, err := os.ReadFile(filepath.Join(dir, "ai-config.json"))
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	if strings.Contains(string(persisted), key) {
		t.Fatalf("persisted config leaked raw API key: %s", persisted)
	}
	if !strings.Contains(string(persisted), "enc:") {
		t.Fatalf("persisted config did not contain ciphertext: %s", persisted)
	}
	info, err := os.Stat(filepath.Join(dir, "ai-config.json"))
	if err != nil {
		t.Fatalf("stat persisted config: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("persisted config mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestAIConfigServiceUsesEnvironmentOnlyWithoutPersistedConfig(t *testing.T) {
	t.Setenv("NRCC_AI_ENABLED", "true")
	t.Setenv("NRCC_AI_PROVIDER", "openai")
	t.Setenv("NRCC_AI_ENDPOINT", "https://env.example.test/v1")
	t.Setenv("NRCC_AI_MODEL", "env-model")
	t.Setenv("NRCC_AI_API_KEY", "env-key")

	svc := NewAIConfigService(t.TempDir(), "encryption-key")
	cfg := loadAIConfig(t, svc)
	if cfg.Endpoint != "https://env.example.test/v1" || cfg.APIKey != "env-key" {
		t.Fatalf("Load() did not use environment fallback: %#v", cfg)
	}
}

func TestAIConfigServicePersistedConfigWinsOverEnvironment(t *testing.T) {
	t.Setenv("NRCC_AI_ENABLED", "false")
	t.Setenv("NRCC_AI_PROVIDER", "offline")
	t.Setenv("NRCC_AI_MODEL", "env-model")

	svc := NewAIConfigService(t.TempDir(), "encryption-key")
	saveAIConfig(t, svc, AIConfig{Enabled: true, Provider: "offline", Model: "persisted-model", APIKey: "persisted-key"})
	cfg := loadAIConfig(t, svc)
	if !cfg.Enabled || cfg.Provider != "offline" || cfg.Model != "persisted-model" || cfg.APIKey != "persisted-key" {
		t.Fatalf("persisted config did not win: %#v", cfg)
	}
	saveAIConfig(t, svc, AIConfig{Provider: "offline", Model: "updated-model"})
	cfg = loadAIConfig(t, svc)
	if cfg.Enabled || cfg.Model != "updated-model" || cfg.APIKey != "persisted-key" {
		t.Fatalf("update did not preserve the write-only API key: %#v", cfg)
	}
}

func TestAIConfigServiceAllowsOfflineWithoutKeyAndRejectsRemoteWithoutEncryption(t *testing.T) {
	svc := NewAIConfigService(t.TempDir(), "")
	if err := svc.Save(AIConfig{Enabled: true, Provider: "offline", Model: "local"}); err != nil {
		t.Fatalf("offline config without key should be valid: %v", err)
	}
	if err := svc.Save(AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "remote", APIKey: "secret"}); !errors.Is(err, ErrAIEncryptionKeyRequired) {
		t.Fatalf("remote API key without encryption key error = %v, want ErrAIEncryptionKeyRequired", err)
	}
}

func TestAIConfigServiceProbeReturnsStableErrors(t *testing.T) {
	tests := []struct {
		name string
		cfg  AIConfig
		want error
	}{
		{"disabled", AIConfig{Provider: "offline"}, ErrAIDisabled},
		{"incomplete", AIConfig{Enabled: true, Provider: "openai", Model: "model", APIKey: "key"}, ErrAIConfigIncomplete},
		{"missing key", AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model"}, ErrAIKeyRequired},
		{"invalid endpoint", AIConfig{Enabled: true, Provider: "openai", Endpoint: "not a URL", Model: "model", APIKey: "key"}, ErrAIInvalidEndpoint},
		{"insecure endpoint", AIConfig{Enabled: true, Provider: "openai", Endpoint: "http://api.example.test", Model: "model", APIKey: "key"}, ErrAIInvalidEndpoint},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAIConfigService(t.TempDir(), "encryption-key")
			saveAIConfig(t, svc, tt.cfg)
			if err := svc.Probe(context.Background()); !errors.Is(err, tt.want) {
				t.Fatalf("Probe() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestAIConfigServiceProbeBoundsInjectedProviderTimeout(t *testing.T) {
	svc := newConfiguredAIService(t, WithAIProbeTimeout(time.Millisecond), WithAIProviderProbe(func(ctx context.Context, _ AIConfig) error {
		<-ctx.Done()
		return ctx.Err()
	}))
	if err := svc.Probe(context.Background()); !errors.Is(err, ErrAIProbeTimeout) {
		t.Fatalf("Probe() error = %v, want ErrAIProbeTimeout", err)
	}
}

func TestAIConfigServiceProbeClassifiesUnreachableProvider(t *testing.T) {
	svc := newConfiguredAIService(t, WithAIProviderProbe(func(context.Context, AIConfig) error {
		return errors.New("connection refused")
	}))
	if err := svc.Probe(context.Background()); !errors.Is(err, ErrAIProbeUnreachable) {
		t.Fatalf("Probe() error = %v, want ErrAIProbeUnreachable", err)
	}
}

func TestAIConfigServiceProbeSucceedsWithInjectedProvider(t *testing.T) {
	svc := newConfiguredAIService(t, WithAIProviderProbe(func(context.Context, AIConfig) error {
		return nil
	}))
	if err := svc.Probe(context.Background()); err != nil {
		t.Fatalf("Probe() error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := svc.Probe(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Probe() error = %v, want context.Canceled", err)
	}
}

func TestAIConfigServiceStatusRequiresPersistedRemoteConnectionResult(t *testing.T) {
	tests := []struct {
		name       string
		cfg        AIConfig
		connection string
		want       string
	}{
		{"disabled", AIConfig{Provider: "offline"}, "", "disabled"},
		{"remote incomplete", AIConfig{Enabled: true, Provider: "openai", Model: "model"}, "", "incomplete"},
		{"remote awaiting test", AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "key"}, "", "incomplete"},
		{"remote testing", AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "key"}, "testing", "testing"},
		{"remote unreachable", AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "key"}, "unreachable", "unreachable"},
		{"offline without key", AIConfig{Enabled: true, Provider: "offline", Model: "local"}, "", "ready"},
		{"remote ready", AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "key"}, "ready", "ready"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probed := false
			svc := NewAIConfigService(t.TempDir(), "encryption-key", WithAIProviderProbe(func(context.Context, AIConfig) error {
				probed = true
				return nil
			}))
			saveAIConfig(t, svc, tt.cfg)
			if tt.connection != "" {
				if err := svc.updateConnection(tt.connection, "test status"); err != nil {
					t.Fatalf("updateConnection() error: %v", err)
				}
			}
			got, err := svc.Status()
			if err != nil {
				t.Fatalf("Status() error: %v", err)
			}
			if got.Status != tt.want {
				t.Errorf("Status().Status = %q, want %q", got.Status, tt.want)
			}
			if probed {
				t.Error("Status() must not probe the provider")
			}
		})
	}
}

func TestAIConfigServiceSaveInvalidatesChangedRemoteConnectionAndPreservesExistingFailure(t *testing.T) {
	svc := newConfiguredAIService(t)
	if err := svc.updateConnection("unreachable", "The provider could not be reached"); err != nil {
		t.Fatalf("updateConnection() error: %v", err)
	}

	// Saving unchanged settings must not turn an unavailable provider into ready.
	saveAIConfig(t, svc, AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model"})
	status, err := svc.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if status.Status != "unreachable" {
		t.Fatalf("Status().Status = %q, want unreachable", status.Status)
	}

	// Changing a remote setting invalidates a prior success until it is retested.
	if err := svc.updateConnection("ready", ""); err != nil {
		t.Fatalf("updateConnection() error: %v", err)
	}
	saveAIConfig(t, svc, AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://other.example.test", Model: "model"})
	status, err = svc.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if status.Status != "incomplete" {
		t.Fatalf("Status().Status = %q, want incomplete", status.Status)
	}
}

func TestAIConfigServiceTestRecordsSafeConnectionStatus(t *testing.T) {
	tests := []struct {
		name       string
		probeErr   error
		wantStatus string
		wantReason string
	}{
		{"ready", nil, "ready", ""},
		{"timeout", context.DeadlineExceeded, "unreachable", "The provider did not respond before the connection test timed out"},
		{"unreachable", errors.New("request failed with Authorization: Bearer secret"), "unreachable", "The provider could not be reached"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAIConfigService(t.TempDir(), "encryption-key", WithAIProviderProbe(func(context.Context, AIConfig) error {
				return tt.probeErr
			}))
			saveAIConfig(t, svc, AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "secret"})

			got, err := svc.Test(context.Background())
			if tt.probeErr != nil && err == nil {
				t.Fatal("Test() error = nil, want probe error")
			}
			if tt.probeErr == nil && err != nil {
				t.Fatalf("Test() error = %v", err)
			}
			if got.Status != tt.wantStatus || got.Reason != tt.wantReason {
				t.Errorf("Test() = %#v, want status %q and reason %q", got, tt.wantStatus, tt.wantReason)
			}
			if strings.Contains(got.Reason, "secret") {
				t.Fatalf("connection reason leaked secret: %q", got.Reason)
			}

			persisted, err := svc.Status()
			if err != nil {
				t.Fatalf("Status() error = %v", err)
			}
			if persisted != got {
				t.Errorf("Status() = %#v, want persisted result %#v", persisted, got)
			}
		})
	}
}
