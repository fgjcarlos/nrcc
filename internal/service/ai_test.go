package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactSecretsRedactsCredentialsEnvAndPayloadSamples(t *testing.T) {
	flow := map[string]interface{}{
		"id":       "tab-1",
		"type":     "tab",
		"name":     "Prod flow",
		"password": "super-secret",
		"credentials": map[string]interface{}{
			"token": "abc123",
		},
		"env": "PUBLIC_VALUE=ok\nAPI_KEY=real-key\nNODE_RED_PASSWORD=real-password",
		"payload": map[string]interface{}{
			"customer": "Jane",
		},
		"nodes": []interface{}{
			map[string]interface{}{
				"id":            "http-1",
				"type":          "http request",
				"authorization": "Bearer shhh",
				"sample":        "private sample",
			},
		},
	}

	redacted := RedactSecrets(flow)
	encoded, err := json.Marshal(redacted)
	if err != nil {
		t.Fatalf("marshal redacted flow: %v", err)
	}
	text := string(encoded)
	for _, secret := range []string{"super-secret", "abc123", "real-key", "real-password", "Bearer shhh", "private sample", "Jane"} {
		if strings.Contains(text, secret) {
			t.Fatalf("redacted output leaked %q: %s", secret, text)
		}
	}
	if !strings.Contains(text, redactedValue) {
		t.Fatalf("expected redaction marker in %s", text)
	}
	if !strings.Contains(text, "PUBLIC_VALUE=ok") {
		t.Fatalf("expected non-secret env value to remain in %s", text)
	}
}

func TestBuildProviderRequestUsesRedactedReviewOnlyMessages(t *testing.T) {
	svc := NewAIService(AIConfig{Enabled: true, Provider: "openai", Endpoint: "https://example.invalid/v1/chat/completions", Model: "test-model"})
	providerReq, err := svc.BuildProviderRequest(AIFlowRequest{
		Action: AIActionGenerate,
		Prompt: "add retry handling",
		Flow: map[string]interface{}{
			"id":        "tab-1",
			"apiKey":    "live-key",
			"payload":   "private payload sample",
			"safeField": "keep-me",
		},
	})
	if err != nil {
		t.Fatalf("BuildProviderRequest returned error: %v", err)
	}
	if providerReq.Provider != "openai" || providerReq.Model != "test-model" {
		t.Fatalf("unexpected provider request metadata: %#v", providerReq)
	}
	if providerReq.Meta["reviewOnly"] != true || providerReq.Meta["redacted"] != true {
		t.Fatalf("expected reviewOnly/redacted meta, got %#v", providerReq.Meta)
	}
	joined := providerReq.Messages[0].Content + providerReq.Messages[1].Content
	for _, leaked := range []string{"live-key", "private payload sample"} {
		if strings.Contains(joined, leaked) {
			t.Fatalf("provider request leaked %q: %s", leaked, joined)
		}
	}
	for _, want := range []string{"review-first", "candidate JSON", redactedValue, "keep-me", "add retry handling"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("provider request missing %q: %s", want, joined)
		}
	}
}

func TestAssistFlowOfflineNoKeyReturnsReviewOnlyCandidate(t *testing.T) {
	svc := NewAIService(AIConfig{Enabled: true, Provider: "offline", Model: "offline"})
	resp, err := svc.AssistFlow(context.Background(), AIFlowRequest{Action: AIActionGenerate, Flow: map[string]interface{}{"token": "secret"}})
	if err != nil {
		t.Fatalf("AssistFlow offline returned error: %v", err)
	}
	if !resp.ReviewOnly || !resp.Redacted || resp.CandidateFlow == nil {
		t.Fatalf("expected review-only redacted candidate response: %#v", resp)
	}
	encoded, _ := json.Marshal(resp.Request)
	if strings.Contains(string(encoded), "secret") {
		t.Fatalf("offline response leaked secret in request: %s", encoded)
	}
}

func TestAssistFlowDisabledRequiresExplicitConfiguration(t *testing.T) {
	svc := NewAIService(AIConfig{Enabled: false, Provider: "offline", Model: "offline"})
	_, err := svc.AssistFlow(context.Background(), AIFlowRequest{Action: AIActionExplain, Flow: map[string]interface{}{"id": "1"}})
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("expected disabled configuration error, got %v", err)
	}
}
