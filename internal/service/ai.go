package service

import (
	"bytes"
	"context"
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

// AIConfig controls optional AI flow assistance. AI is disabled unless explicitly enabled.
type AIConfig struct {
	Enabled  bool
	Provider string
	Endpoint string
	Model    string
	APIKey   string
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
		Enabled:  strings.EqualFold(os.Getenv("NRCC_AI_ENABLED"), "true"),
		Provider: strings.ToLower(provider),
		Endpoint: endpoint,
		Model:    model,
		APIKey:   os.Getenv("NRCC_AI_API_KEY"),
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
	Action AIFlowAction `json:"action"`
	Flow   interface{}  `json:"flow"`
	Prompt string       `json:"prompt,omitempty"`
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
}

// AIProviderRequest is the redacted provider request. Tests assert this is safe before leaving host.
type AIProviderRequest struct {
	Provider string                 `json:"provider"`
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
	return &AIService{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}

func (s *AIService) AssistFlow(ctx context.Context, req AIFlowRequest) (AIFlowResponse, error) {
	if !s.cfg.Enabled {
		return AIFlowResponse{}, errors.New("AI flow assistance is disabled; set NRCC_AI_ENABLED=true to enable")
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

	if s.cfg.Provider == "offline" {
		resp.Summary = offlineSummary(req.Action)
		resp.Suggestions = offlineSuggestions(req.Action)
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

	if s.cfg.APIKey == "" {
		return AIFlowResponse{}, errors.New("AI provider API key is required for non-offline providers")
	}
	if s.cfg.Endpoint == "" {
		return AIFlowResponse{}, errors.New("AI provider endpoint is required")
	}

	body, err := json.Marshal(map[string]interface{}{"model": providerReq.Model, "messages": providerReq.Messages})
	if err != nil {
		return AIFlowResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return AIFlowResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)

	httpResp, err := s.client.Do(httpReq)
	if err != nil {
		return AIFlowResponse{}, err
	}
	defer httpResp.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return AIFlowResponse{}, fmt.Errorf("AI provider returned %s", httpResp.Status)
	}
	resp.Summary = strings.TrimSpace(string(payload))
	return resp, nil
}

func (s *AIService) BuildProviderRequest(req AIFlowRequest) (AIProviderRequest, error) {
	if !validAIAction(req.Action) {
		return AIProviderRequest{}, fmt.Errorf("unsupported AI flow action: %s", req.Action)
	}
	redactedFlow := RedactSecrets(req.Flow)
	flowJSON, err := json.MarshalIndent(redactedFlow, "", "  ")
	if err != nil {
		return AIProviderRequest{}, fmt.Errorf("failed to encode redacted flow: %w", err)
	}
	system := "You are NRCC's Node-RED flow copilot. Never ask to auto-apply changes. Return review-first output; generated flows must be explicit candidate JSON for a human to inspect and import/apply manually."
	user := fmt.Sprintf("Action: %s\nUser prompt: %s\nRedacted Node-RED flow JSON:\n%s", req.Action, req.Prompt, string(flowJSON))
	return AIProviderRequest{
		Provider: s.cfg.Provider,
		Endpoint: s.cfg.Endpoint,
		Model:    s.cfg.Model,
		Messages: []AIMessage{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Meta:     map[string]interface{}{"reviewOnly": true, "redacted": true},
	}, nil
}

func validAIAction(action AIFlowAction) bool {
	switch action {
	case AIActionExplain, AIActionSuggest, AIActionAudit, AIActionGenerate:
		return true
	default:
		return false
	}
}

func offlineSummary(action AIFlowAction) string {
	switch action {
	case AIActionExplain:
		return "Offline AI mode: redacted request constructed for flow explanation. Configure a provider to receive model output."
	case AIActionSuggest:
		return "Offline AI mode: redacted request constructed for improvement suggestions. Configure a provider to receive model output."
	case AIActionAudit:
		return "Offline AI mode: redacted request constructed for flow audit. Configure a provider to receive model output."
	case AIActionGenerate:
		return "Offline AI mode: redacted request constructed for candidate generation. Candidate output requires human review and manual import/apply."
	default:
		return "Offline AI mode: redacted request constructed."
	}
}

func offlineSuggestions(action AIFlowAction) []string {
	return []string{"AI requests are disabled unless NRCC_AI_ENABLED=true.", "Secrets are redacted before provider request construction.", "Review any generated candidate flow JSON before importing or applying it."}
}

var secretKeyPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|credential|credentials|apikey|api_key|access[_-]?key|refresh[_-]?token|private[_-]?key|authorization|bearer|cookie|session)`)
var envSecretLinePattern = regexp.MustCompile(`(?im)^([A-Z0-9_]*(?:PASSWORD|PASSWD|SECRET|TOKEN|API_KEY|ACCESS_KEY|PRIVATE_KEY|COOKIE|SESSION)[A-Z0-9_]*\s*=\s*)([^\n\r]+)`)

// RedactSecrets recursively redacts credentials, environment values, and payload samples.
func RedactSecrets(v interface{}) interface{} {
	switch typed := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for k, val := range typed {
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
		return redactSecretString(typed)
	default:
		return typed
	}
}

func redactSecretString(s string) string {
	if strings.Contains(s, "=") {
		return envSecretLinePattern.ReplaceAllString(s, "${1}"+redactedValue)
	}
	if strings.Contains(strings.ToLower(s), "bearer ") {
		return regexp.MustCompile(`(?i)bearer\s+[^\s,;]+`).ReplaceAllString(s, "Bearer "+redactedValue)
	}
	return s
}
