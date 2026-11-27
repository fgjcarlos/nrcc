package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrAIProviderAuth   = errors.New("AI provider rejected credentials")
	ErrAIProviderServer = errors.New("AI provider failed")
	ErrAITimeout        = errors.New("AI provider request timed out")
)

// Cleartext is only for explicitly opted-in literal LAN addresses, never via a proxy.
var aiDirectTransport = &http.Transport{DisableKeepAlives: true}

type providerPayload struct {
	Summary       string          `json:"summary"`
	Suggestions   []string        `json:"suggestions"`
	AuditFindings []string        `json:"auditFindings"`
	CandidateFlow json.RawMessage `json:"candidateFlow"`
}

func parseProviderResponse(provider string, raw []byte) (providerPayload, error) {
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
		Response string `json:"response"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return providerPayload{}, errors.New("invalid AI provider response")
	}
	text := ""
	switch provider {
	case "openai":
		if len(envelope.Choices) > 0 {
			text = envelope.Choices[0].Message.Content
		}
	case "anthropic":
		blocks := envelope.Content
		if len(blocks) == 0 && len(envelope.Message.Content) > 0 {
			_ = json.Unmarshal(envelope.Message.Content, &blocks)
		}
		for _, b := range blocks {
			if b.Type == "text" {
				text += b.Text
			}
		}
	case "ollama":
		if len(envelope.Message.Content) > 0 {
			_ = json.Unmarshal(envelope.Message.Content, &text)
		}
		if text == "" {
			var content struct {
				Content string `json:"content"`
			}
			if json.Unmarshal(envelope.Message.Content, &content) == nil {
				text = content.Content
			}
		}
		if text == "" {
			text = envelope.Response
		}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return providerPayload{}, errors.New("empty AI provider response")
	}
	structured := text
	if strings.HasPrefix(text, "```") && strings.HasSuffix(text, "```") {
		if start := strings.IndexByte(text, '\n'); start >= 0 {
			structured = strings.TrimSuffix(text[start+1:], "```")
		}
	}
	var out providerPayload
	if err := json.Unmarshal([]byte(structured), &out); err == nil && strings.TrimSpace(out.Summary) != "" {
		return out, nil
	}
	out = providerPayload{Summary: text}
	return out, nil
}

func providerTarget(cfg AIConfig) (string, error) {
	if cfg.Provider == "offline" {
		return "", nil
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Host == "" {
		return "", ErrAIInvalidEndpoint
	}
	path := map[string]string{"openai": "/v1/chat/completions", "anthropic": "/v1/messages", "ollama": "/api/chat"}[cfg.Provider]
	if path == "" {
		return cfg.Endpoint, nil
	}
	current := strings.TrimRight(u.Path, "/")
	if strings.HasSuffix(current, path) {
		return u.String(), nil
	}
	if current != "" && !strings.HasSuffix(current, "/v1") {
		return u.String(), nil
	}
	if strings.HasSuffix(current, "/v1") {
		path = strings.TrimPrefix(path, "/v1")
	}
	u.Path = current + path
	return u.String(), nil
}

func buildProviderHTTP(cfg AIConfig, messages []AIMessage) (string, map[string]interface{}, map[string]string, error) {
	target, err := providerTarget(cfg)
	if err != nil {
		return "", nil, nil, err
	}
	body, _ := providerRequestBody(cfg.Provider, messages, cfg.Model)
	headers := map[string]string{"Content-Type": "application/json"}
	if cfg.Provider == "anthropic" {
		headers["x-api-key"] = cfg.APIKey
		headers["anthropic-version"] = "2023-06-01"
	} else if cfg.Provider != "ollama" {
		headers["Authorization"] = "Bearer " + cfg.APIKey
	}
	return target, body, headers, nil
}

func providerRequestBody(provider string, messages []AIMessage, model string) (map[string]interface{}, map[string]interface{}) {
	ms := make([]map[string]string, 0, len(messages))
	system := ""
	for _, m := range messages {
		if provider == "anthropic" && m.Role == "system" {
			system = m.Content
			continue
		}
		ms = append(ms, map[string]string{"role": m.Role, "content": m.Content})
	}
	switch provider {
	case "offline":
		return nil, nil
	case "anthropic":
		body := map[string]interface{}{"model": model, "max_tokens": 4096, "messages": ms}
		if system != "" {
			body["system"] = system
		}
		return body, nil
	case "ollama":
		return map[string]interface{}{"model": model, "messages": ms, "stream": false}, nil
	default:
		return map[string]interface{}{"model": model, "messages": ms}, nil
	}
}

func validAIEndpointFor(cfg AIConfig) bool {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Hostname() == "" || u.User != nil || u.ForceQuery || u.RawQuery != "" || strings.Contains(cfg.Endpoint, "#") {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" || cfg.Provider != "ollama" || !cfg.AllowPrivateHTTP {
		return false
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

func validateCandidate(raw json.RawMessage) (map[string]interface{}, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("invalid candidate flow")
	}
	var nodes []interface{}
	switch x := v.(type) {
	case []interface{}:
		nodes = x
	case map[string]interface{}:
		if n, ok := x["nodes"].([]interface{}); ok {
			nodes = n
		} else if n, ok := x["flow"].([]interface{}); ok {
			nodes = n
		} else {
			return nil, errors.New("invalid candidate flow")
		}
	default:
		return nil, errors.New("invalid candidate flow")
	}
	for _, node := range nodes {
		obj, ok := node.(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid candidate flow")
		}
		id, idOK := obj["id"].(string)
		typ, typeOK := obj["type"].(string)
		if !idOK || strings.TrimSpace(id) == "" || !typeOK || strings.TrimSpace(typ) == "" {
			return nil, errors.New("invalid candidate flow")
		}
	}
	return map[string]interface{}{"nodes": nodes}, nil
}
