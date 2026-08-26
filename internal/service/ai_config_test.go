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

func TestAIConfigServicePersistsEncryptedSecretAndReturnsRedactedView(t *testing.T) {
	dir := t.TempDir()
	svc := NewAIConfigService(dir, "encryption-key")
	key := "test-api-key-must-not-be-persisted-raw"

	if err := svc.Save(AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test/v1", Model: "test-model", APIKey: key}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

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
	cfg, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Endpoint != "https://env.example.test/v1" || cfg.APIKey != "env-key" {
		t.Fatalf("Load() did not use environment fallback: %#v", cfg)
	}
}

func TestAIConfigServicePersistedConfigWinsOverEnvironment(t *testing.T) {
	t.Setenv("NRCC_AI_ENABLED", "false")
	t.Setenv("NRCC_AI_PROVIDER", "offline")
	t.Setenv("NRCC_AI_MODEL", "env-model")

	svc := NewAIConfigService(t.TempDir(), "encryption-key")
	if err := svc.Save(AIConfig{Enabled: true, Provider: "offline", Model: "persisted-model"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	cfg, err := svc.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.Enabled || cfg.Provider != "offline" || cfg.Model != "persisted-model" {
		t.Fatalf("persisted config did not win: %#v", cfg)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAIConfigService(t.TempDir(), "encryption-key")
			if err := svc.Save(tt.cfg); err != nil {
				t.Fatalf("Save() error: %v", err)
			}
			if err := svc.Probe(context.Background()); !errors.Is(err, tt.want) {
				t.Fatalf("Probe() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestAIConfigServiceProbeBoundsInjectedProviderTimeout(t *testing.T) {
	svc := NewAIConfigService(t.TempDir(), "encryption-key", WithAIProbeTimeout(time.Millisecond), WithAIProviderProbe(func(ctx context.Context, _ AIConfig) error {
		<-ctx.Done()
		return ctx.Err()
	}))
	if err := svc.Save(AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "key"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if err := svc.Probe(context.Background()); !errors.Is(err, ErrAIProbeTimeout) {
		t.Fatalf("Probe() error = %v, want ErrAIProbeTimeout", err)
	}
}

func TestAIConfigServiceProbeClassifiesUnreachableProvider(t *testing.T) {
	svc := NewAIConfigService(t.TempDir(), "encryption-key", WithAIProviderProbe(func(context.Context, AIConfig) error {
		return errors.New("connection refused")
	}))
	if err := svc.Save(AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "key"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if err := svc.Probe(context.Background()); !errors.Is(err, ErrAIProbeUnreachable) {
		t.Fatalf("Probe() error = %v, want ErrAIProbeUnreachable", err)
	}
}

func TestAIConfigServiceProbeSucceedsWithInjectedProvider(t *testing.T) {
	svc := NewAIConfigService(t.TempDir(), "encryption-key", WithAIProviderProbe(func(context.Context, AIConfig) error {
		return nil
	}))
	if err := svc.Save(AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://api.example.test", Model: "model", APIKey: "key"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if err := svc.Probe(context.Background()); err != nil {
		t.Fatalf("Probe() error: %v", err)
	}
}
