package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const redactedValue = "[REDACTED]"

// Sentinels for AI service failures that the HTTP layer translates to
// specific status codes. Use errors.Is to match — never compare err.Error()
// strings; reworded messages silently change HTTP responses.
var (
	ErrAIDisabled       = errors.New("AI flow assistance is disabled; set NRCC_AI_ENABLED=true to enable")
	ErrAIKeyRequired    = errors.New("AI provider API key is required for non-offline providers")
	ErrAIEndpointReqd   = errors.New("AI provider endpoint is required")
	ErrAITimeoutInvalid = errors.New("AI analysis timeout must be between 1 and 300 seconds")
)

// AIConfig controls optional AI flow assistance. AI is disabled unless explicitly enabled.
type AIConfig struct {
	Enabled          bool
	Provider         string
	Endpoint         string
	Model            string
	APIKey           string
	AllowPrivateHTTP bool
	TimeoutSeconds   int
}

// LoadAIConfigFromEnv loads AI configuration from environment variables.
func LoadAIConfigFromEnv() AIConfig {
	provider := strings.TrimSpace(os.Getenv("NRCC_AI_PROVIDER"))
	if provider == "" {
		provider = "offline"
	}
	endpoint := strings.TrimSpace(os.Getenv("NRCC_AI_ENDPOINT"))
	if endpoint == "" && provider == "openai" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	model := strings.TrimSpace(os.Getenv("NRCC_AI_MODEL"))
	if model == "" {
		model = "gpt-4o-mini"
	}
	return AIConfig{
		TimeoutSeconds: 30,
		Enabled:        strings.EqualFold(os.Getenv("NRCC_AI_ENABLED"), "true"),
		Provider:       strings.ToLower(provider),
		Endpoint:       endpoint,
		Model:          model,
		APIKey:         os.Getenv("NRCC_AI_API_KEY"),
	}
}

// AIFlowAction is a supported review-first AI flow operation.
type AIFlowAction string

const (
	AIActionExplain  AIFlowAction = "explain"
	AIActionSuggest  AIFlowAction = "suggest"
	AIActionAudit    AIFlowAction = "audit"
	AIActionGenerate AIFlowAction = "generate"
)

// AIFlowRequest is the API request payload for AI flow assistance.
type AIFlowRequest struct {
	Action       AIFlowAction `json:"action"`
	Flow         interface{}  `json:"flow"`
	Prompt       string       `json:"prompt,omitempty"`
	PreviewOnly  bool         `json:"previewOnly,omitempty"`
	PreviewToken string       `json:"previewToken,omitempty"`
}

// AIFlowResponse is a review-first AI response. CandidateFlow is never applied server-side.
type AIFlowResponse struct {
	Enabled       bool                   `json:"enabled"`
	Provider      string                 `json:"provider"`
	Action        AIFlowAction           `json:"action"`
	ReviewOnly    bool                   `json:"reviewOnly"`
	Redacted      bool                   `json:"redacted"`
	Summary       string                 `json:"summary"`
	Suggestions   []string               `json:"suggestions,omitempty"`
	AuditFindings []string               `json:"auditFindings,omitempty"`
	CandidateFlow map[string]interface{} `json:"candidateFlow,omitempty"`
	Request       AIProviderRequest      `json:"request,omitempty"`
	PreviewToken  string                 `json:"previewToken,omitempty"`
}

// AIProviderRequest is the redacted provider request. Tests assert this is safe before leaving host.
type AIProviderRequest struct {
	Provider string                 `json:"provider"`
	Body     map[string]interface{} `json:"body,omitempty"`
	Endpoint string                 `json:"endpoint,omitempty"`
	Model    string                 `json:"model"`
	Messages []AIMessage            `json:"messages"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIService constructs and optionally dispatches redacted AI flow requests.
type AIService struct {
	cfg    AIConfig
	client *http.Client
}

func NewAIService(cfg AIConfig) *AIService {
	cfg = normalizeAIConfig(cfg)
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 30
	}
	return &AIService{cfg: cfg, client: &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (s *AIService) AssistFlow(ctx context.Context, req AIFlowRequest) (AIFlowResponse, error) {
	if !s.cfg.Enabled {
		return AIFlowResponse{}, ErrAIDisabled
	}
	providerReq, err := s.BuildProviderRequest(req)
	if err != nil {
		return AIFlowResponse{}, err
	}

	resp := AIFlowResponse{
		Enabled:    true,
		Provider:   providerReq.Provider,
		Action:     req.Action,
		ReviewOnly: true,
		Redacted:   true,
		Request:    providerReq,
	}

	if req.PreviewOnly {
		resp.PreviewToken = previewToken(providerReq)
		return resp, nil
	}
	if req.PreviewToken != "" && req.PreviewToken != previewToken(providerReq) {
		return AIFlowResponse{}, ErrAIPreviewChanged
	}
	if s.cfg.Provider == "offline" {
		switch req.Action {
		case AIActionExplain:
			resp.Summary = "Offline AI mode: redacted request constructed for flow explanation. Configure a provider to receive model output."
		case AIActionSuggest:
			resp.Summary = "Offline AI mode: redacted request constructed for improvement suggestions. Configure a provider to receive model output."
		case AIActionAudit:
			resp.Summary = "Offline AI mode: redacted request constructed for flow audit. Configure a provider to receive model output."
		case AIActionGenerate:
			resp.Summary = "Offline AI mode: redacted request constructed for candidate generation. Candidate output requires human review and manual import/apply."
		default:
			resp.Summary = "Offline AI mode: redacted request constructed."
		}
		resp.Suggestions = []string{"AI requests are disabled unless NRCC_AI_ENABLED=true.", "Secrets are redacted before provider request construction.", "Review any generated candidate flow JSON before importing or applying it."}
		if req.Action == AIActionAudit {
			resp.AuditFindings = []string{"Review credential nodes, debug nodes, and external HTTP endpoints before import/apply."}
		}
		if req.Action == AIActionGenerate {
			resp.CandidateFlow = map[string]interface{}{
				"reviewRequired": true,
				"nodes":          []interface{}{},
				"note":           "Offline mode returns a placeholder candidate only; no changes are applied automatically.",
			}
		}
		return resp, nil
	}

	if s.cfg.APIKey == "" && s.cfg.Provider != "ollama" {
		return AIFlowResponse{}, ErrAIKeyRequired
	}
	if s.cfg.Endpoint == "" {
		return AIFlowResponse{}, ErrAIEndpointReqd
	}
	target, requestBody, headers, err := buildProviderHTTP(s.cfg, providerReq.Messages)
	if err != nil {
		return AIFlowResponse{}, err
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return AIFlowResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return AIFlowResponse{}, err
	}
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}

	if !validAIEndpointFor(s.cfg) {
		return AIFlowResponse{}, ErrAIInvalidEndpoint
	}
	client := *s.client
	if httpReq.URL.Scheme == "http" {
		client.Transport = aiDirectTransport
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) && ctx.Err() == context.DeadlineExceeded {
			return AIFlowResponse{}, ErrAITimeout
		}
		return AIFlowResponse{}, ErrAIProviderServer
	}
	defer func() { _ = httpResp.Body.Close() }()
	if httpResp.StatusCode == http.StatusUnauthorized || httpResp.StatusCode == http.StatusForbidden {
		return AIFlowResponse{}, ErrAIProviderAuth
	}
	if httpResp.StatusCode >= 500 {
		return AIFlowResponse{}, ErrAIProviderServer
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return AIFlowResponse{}, ErrAIProviderServer
	}
	payload, readErr := io.ReadAll(io.LimitReader(httpResp.Body, (4<<20)+1))
	if readErr != nil {
		if errors.Is(readErr, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return AIFlowResponse{}, ErrAITimeout
		}
		return AIFlowResponse{}, ErrAIProviderServer
	}
	if len(payload) > 4<<20 {
		return AIFlowResponse{}, ErrAIProviderServer
	}
	parsed, err := parseProviderResponse(s.cfg.Provider, payload)
	if err != nil {
		return AIFlowResponse{}, ErrAIProviderServer
	}
	resp.Summary, resp.Suggestions, resp.AuditFindings = parsed.Summary, parsed.Suggestions, parsed.AuditFindings
	resp.CandidateFlow, err = validateCandidate(parsed.CandidateFlow)
	if err != nil {
		return AIFlowResponse{}, fmt.Errorf("%w: candidate flow rejected", ErrAIProviderServer)
	}
	return resp, nil
}

var ErrAIPreviewChanged = errors.New("AI_PREVIEW_CHANGED")

func previewToken(req AIProviderRequest) string {
	b, _ := json.Marshal(req)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func (s *AIService) BuildProviderRequest(req AIFlowRequest) (AIProviderRequest, error) {
	switch req.Action {
	case AIActionExplain, AIActionSuggest, AIActionAudit, AIActionGenerate:
	default:
		return AIProviderRequest{}, fmt.Errorf("unsupported AI flow action: %s", req.Action)
	}
	redactedFlow := RedactSecrets(req.Flow)
	flowJSON, err := json.MarshalIndent(redactedFlow, "", "  ")
	if err != nil {
		return AIProviderRequest{}, fmt.Errorf("failed to encode redacted flow: %w", err)
	}
	system := "You are NRCC's Node-RED flow copilot. Never ask to auto-apply changes. Return review-first output; generated flows must be explicit candidate JSON for a human to inspect and import/apply manually. For generate, return JSON shaped exactly as {\"summary\":\"...\",\"candidateFlow\":{\"nodes\":[...]}}."
	redactedPrompt, _ := RedactSecrets(req.Prompt).(string)
	user := fmt.Sprintf("Action: %s\nUser prompt: %s\nRedacted Node-RED flow JSON:\n%s", req.Action, redactedPrompt, string(flowJSON))
	messages := []AIMessage{{Role: "system", Content: system}, {Role: "user", Content: user}}
	target, body, _, err := buildProviderHTTP(s.cfg, messages)
	if err != nil {
		return AIProviderRequest{}, err
	}
	return AIProviderRequest{Provider: s.cfg.Provider, Endpoint: target, Model: s.cfg.Model, Messages: messages, Body: body, Meta: map[string]interface{}{"reviewOnly": true, "redacted": true}}, nil
}

// secretKeyPattern matches the property names that carry a secret. The
// TLS private-key passphrase ("passphrase") is part of the set: Node-RED
// stores it next to the certificate material, so it must never leave the
// host inside an AI request (issue #877).
var secretKeyPattern = regexp.MustCompile(`(?i)(password|passwd|passphrase|secret|token|credential|credentials|apikey|api_key|access[_-]?key|refresh[_-]?token|private[_-]?key|authorization|bearer|cookie|session)`)
var envSecretLinePattern = regexp.MustCompile(`(?im)^([A-Z0-9_]*(?:PASSWORD|PASSWD|PASSPHRASE|SECRET|TOKEN|API_KEY|ACCESS_KEY|PRIVATE_KEY|COOKIE|SESSION)[A-Z0-9_]*\s*=\s*)([^\n\r]+)`)

// RedactSecrets recursively redacts credentials, environment values, and payload samples.
func RedactSecrets(v interface{}) interface{} {
	switch typed := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		rowSecret := false
		for _, key := range []string{"name", "key"} {
			if name, ok := typed[key].(string); ok && secretKeyPattern.MatchString(name) {
				rowSecret = true
			}
		}
		for k, val := range typed {
			if rowSecret && strings.EqualFold(k, "value") {
				out[k] = redactedValue
				continue
			}
			if secretKeyPattern.MatchString(k) || strings.EqualFold(k, "payload") || strings.EqualFold(k, "sample") || strings.EqualFold(k, "examplePayload") {
				out[k] = redactedValue
				continue
			}
			out[k] = RedactSecrets(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(typed))
		for i, val := range typed {
			out[i] = RedactSecrets(val)
		}
		return out
	case string:
		if strings.Contains(typed, "=") {
			return envSecretLinePattern.ReplaceAllString(typed, "${1}"+redactedValue)
		}
		if strings.Contains(strings.ToLower(typed), "bearer ") {
			return regexp.MustCompile(`(?i)bearer\s+[^\s,;]+`).ReplaceAllString(typed, "Bearer "+redactedValue)
		}
		return typed
	default:
		return typed
	}
}
