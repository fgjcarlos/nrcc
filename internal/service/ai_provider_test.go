package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAssistFlowProviderRequestsAndResponses(t *testing.T) {
	cases := []struct {
		provider, path string
		status         int
		wantHeader     string
		response       func() string
	}{
		{"openai", "/v1/chat/completions", http.StatusOK, "Bearer synthetic-key", func() string {
			return `{"choices":[{"message":{"content":"{\"summary\":\"openai summary\",\"candidateFlow\":[{\"id\":\"n1\",\"type\":\"inject\"}] }"}}]}`
		}},
		{"anthropic", "/v1/messages", http.StatusOK, "synthetic-key", func() string { return `{"content":[{"type":"text","text":"{\"summary\":\"anthropic summary\"}"}]}` }},
		{"ollama", "/api/chat", http.StatusOK, "", func() string { return `{"message":{"content":"{\"summary\":\"ollama summary\"}"}}` }},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.path {
					t.Errorf("path=%s want %s", r.URL.Path, tc.path)
				}
				if got := r.Header.Get("Authorization"); tc.provider == "openai" && got != tc.wantHeader {
					t.Errorf("authorization=%q", got)
				}
				if tc.provider == "anthropic" && r.Header.Get("x-api-key") != tc.wantHeader {
					t.Errorf("x-api-key=%q", r.Header.Get("x-api-key"))
				}
				if tc.provider == "ollama" && (r.Header.Get("Authorization") != "" || r.Header.Get("x-api-key") != "") {
					t.Error("Ollama received credentials")
				}
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["model"] != "test-model" {
					t.Errorf("provider body=%v", body)
				}
				switch tc.provider {
				case "anthropic":
					if body["system"] == nil || body["messages"] == nil {
						t.Errorf("Anthropic body=%v", body)
					}
				case "ollama":
					if body["stream"] != false || body["messages"] == nil {
						t.Errorf("Ollama body=%v", body)
					}
				case "openai":
					if body["messages"] == nil {
						t.Errorf("OpenAI body=%v", body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.response()))
			}))
			defer server.Close()
			cfg := AIConfig{Enabled: true, Provider: tc.provider, Endpoint: server.URL, Model: "test-model", APIKey: "synthetic-key", AllowPrivateHTTP: tc.provider == "ollama"}
			svc := NewAIService(cfg)
			svc.client.Transport = server.Client().Transport
			resp, err := svc.AssistFlow(context.Background(), AIFlowRequest{Action: AIActionGenerate, Flow: map[string]interface{}{"nodes": []interface{}{}}})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(resp.Summary, tc.provider+" summary") {
				t.Fatalf("summary=%q", resp.Summary)
			}
			if tc.provider == "openai" && len(resp.CandidateFlow["nodes"].([]interface{})) != 1 {
				t.Fatalf("candidate=%#v", resp.CandidateFlow)
			}
		})
	}
}

func TestProviderTargetAppendsCompletionsToV1BasePath(t *testing.T) {
	got, err := providerTarget(AIConfig{Provider: "openai", Endpoint: "https://openrouter.ai/api/v1"})
	if err != nil || got != "https://openrouter.ai/api/v1/chat/completions" {
		t.Fatalf("providerTarget() = %q, %v", got, err)
	}
}

func TestAssistFlowAnalysisUsesConfiguredTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := time.NewTimer(1500 * time.Millisecond)
		defer timer.Stop()
		<-timer.C
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"summary\":\"late\"}"}}]}`))
	}))
	defer server.Close()
	svc := NewAIService(AIConfig{Enabled: true, Provider: "ollama", Endpoint: server.URL, Model: "m", AllowPrivateHTTP: true, TimeoutSeconds: 1})
	_, err := svc.AssistFlow(context.Background(), AIFlowRequest{Action: AIActionExplain, Flow: map[string]interface{}{}})
	if !errors.Is(err, ErrAITimeout) {
		t.Fatalf("AssistFlow() error = %v, want ErrAITimeout", err)
	}
}

func TestAssistFlowHandlesUpstreamBodyFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want error
	}{
		{"read timeout", "", ErrAITimeout},
		{"malformed", `not JSON: synthetic private data`, ErrAIProviderServer},
		{"oversized", `{"message":{"content":"valid prefix"}}` + strings.Repeat(" ", 4<<20), ErrAIProviderServer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.body == "" {
					w.WriteHeader(http.StatusOK)
					w.(http.Flusher).Flush()
					time.Sleep(3 * time.Second)
					return
				}
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			svc := NewAIService(AIConfig{Enabled: true, Provider: "ollama", Endpoint: server.URL, Model: "m", AllowPrivateHTTP: true, TimeoutSeconds: 1})
			got, err := svc.AssistFlow(context.Background(), AIFlowRequest{Action: AIActionExplain})
			if err != tc.want || got.Summary != "" || got.CandidateFlow != nil {
				t.Fatalf("response=%+v error=%v; want empty response and %v", got, err, tc.want)
			}
		})
	}
}

func TestAssistFlowMapsProviderFailuresWithoutEchoingBody(t *testing.T) {
	for _, tc := range []struct {
		code int
		want error
	}{{http.StatusUnauthorized, ErrAIProviderAuth}, {http.StatusInternalServerError, ErrAIProviderServer}} {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte("synthetic upstream secret body"))
			}))
			defer server.Close()
			svc := NewAIService(AIConfig{Enabled: true, Provider: "openai", Endpoint: server.URL, Model: "m", APIKey: "synthetic-key"})
			svc.client.Transport = server.Client().Transport
			_, err := svc.AssistFlow(context.Background(), AIFlowRequest{Action: AIActionExplain, Flow: map[string]interface{}{}})
			if err != tc.want {
				t.Fatalf("error=%v want %v", err, tc.want)
			}
			if strings.Contains(err.Error(), "synthetic") {
				t.Fatalf("unsafe error: %v", err)
			}
		})
	}
}

func TestAssistFlowDoesNotFollowProviderRedirect(t *testing.T) {
	called := make(chan struct{}, 1)
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called <- struct{}{}
		_, _ = io.WriteString(w, `{"message":{"content":"unexpected"}}`)
	}))
	defer destination.Close()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer provider.Close()
	svc := NewAIService(AIConfig{Enabled: true, Provider: "ollama", Endpoint: provider.URL, Model: "m", AllowPrivateHTTP: true})
	if _, err := svc.AssistFlow(context.Background(), AIFlowRequest{Action: AIActionExplain}); err != ErrAIProviderServer {
		t.Fatalf("error=%v, want %v", err, ErrAIProviderServer)
	}
	select {
	case <-called:
		t.Fatal("redirect received a provider request")
	default:
	}
}

func TestPreviewTokenTargetsExactProviderBodyWithoutPreviewNetworkCall(t *testing.T) {
	calls := 0
	var executionBody []byte
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		executionBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"summary\":\"done\"}"}}]}`))
	}))
	defer server.Close()
	svc := NewAIService(AIConfig{Enabled: true, Provider: "openai", Endpoint: server.URL, Model: "m", APIKey: "synthetic-key"})
	svc.client.Transport = server.Client().Transport
	request := AIFlowRequest{Action: AIActionExplain, Flow: map[string]interface{}{"nodes": []interface{}{}}, PreviewOnly: true}
	preview, err := svc.AssistFlow(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("preview contacted provider")
	}
	if !strings.HasSuffix(preview.Request.Endpoint, "/v1/chat/completions") {
		t.Fatalf("target=%s", preview.Request.Endpoint)
	}
	previewBody, _ := json.Marshal(preview.Request.Body)
	request.PreviewOnly = false
	request.PreviewToken = preview.PreviewToken
	if _, err = svc.AssistFlow(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || !bytes.Equal(previewBody, executionBody) {
		t.Fatalf("preview body differs from executed body: preview=%s actual=%s calls=%d", previewBody, executionBody, calls)
	}
	request.Prompt = "changed"
	if _, err = svc.AssistFlow(context.Background(), request); !errors.Is(err, ErrAIPreviewChanged) {
		t.Fatalf("error=%v want preview changed", err)
	}
	if calls != 1 {
		t.Fatalf("changed preview contacted provider: %d", calls)
	}
}

func TestAIEndpointPrivateHTTPRequiresOptInAndLiteralPrivateAddress(t *testing.T) {
	for _, tc := range []struct {
		url         string
		allow, want bool
	}{{"http://127.0.0.1:11434", true, true}, {"http://[::1]:11434", true, true}, {"http://192.168.1.2:11434", true, true}, {"http://localhost:11434", true, false}, {"http://example.test", true, false}, {"http://127.0.0.1", false, false}, {"https://host.test/path#fragment", false, false}, {"https://host.test/path?", false, false}} {
		if got := (validAIEndpointFor(AIConfig{Provider: "ollama", Endpoint: tc.url, AllowPrivateHTTP: tc.allow})); got != tc.want {
			t.Errorf("validAIEndpointFor(%q)=%v want %v", tc.url, got, tc.want)
		}
	}
}

func TestValidateCandidateRequiresNodeREDNodeIdentity(t *testing.T) {
	for _, raw := range []string{`[{"type":"inject"}]`, `[{"id":"n1"}]`, `[{}]`} {
		if _, err := validateCandidate(json.RawMessage(raw)); err == nil {
			t.Errorf("accepted candidate %s", raw)
		}
	}
	if _, err := validateCandidate(json.RawMessage(`[{"id":"n1","type":"inject"}]`)); err != nil {
		t.Fatal(err)
	}
}
