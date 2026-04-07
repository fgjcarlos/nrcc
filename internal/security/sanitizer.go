package security

import (
	"regexp"
	"strings"
)

// Sanitizer redacts sensitive data from maps and strings
type Sanitizer struct{}

// NewSanitizer returns a new Sanitizer
func NewSanitizer() *Sanitizer {
	return &Sanitizer{}
}

// SanitizeMap redacts values for sensitive keys in a map (recursive)
// Sensitive keys: password, secret, token, key, cookie, session, auth, credential, pass
// Redaction value: "[REDACTED]"
func (s *Sanitizer) SanitizeMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return m
	}

	sensitiveKeys := map[string]bool{
		"password":   true,
		"secret":     true,
		"token":      true,
		"key":        true,
		"cookie":     true,
		"session":    true,
		"auth":       true,
		"credential": true,
		"pass":       true,
	}

	result := make(map[string]interface{})
	for k, v := range m {
		lowerKey := strings.ToLower(k)
		if sensitiveKeys[lowerKey] {
			result[k] = "[REDACTED]"
			continue
		}

		// Recursively handle nested maps
		if nestedMap, ok := v.(map[string]interface{}); ok {
			result[k] = s.SanitizeMap(nestedMap)
		} else {
			result[k] = v
		}
	}

	return result
}

// SanitizeString removes potential secrets from a string using regex patterns
func (s *Sanitizer) SanitizeString(input string) string {
	if input == "" {
		return input
	}

	// Pattern 1: API keys (format: "api_key":"...", "apiKey":"...", etc.)
	re1 := regexp.MustCompile(`("(?:api_?key|secret|token|password|auth|credential)"?\s*:\s*)"[^"]*"`)
	input = re1.ReplaceAllString(input, `$1"[REDACTED]"`)

	// Pattern 2: Bearer tokens in Authorization headers
	re2 := regexp.MustCompile(`(Bearer|Token)\s+[A-Za-z0-9\-._~+/]+=*`)
	input = re2.ReplaceAllString(input, "$1 [REDACTED]")

	// Pattern 3: Password patterns in URLs (user:password@)
	re3 := regexp.MustCompile(`://[^:]+:[^@]+@`)
	input = re3.ReplaceAllString(input, "://[REDACTED]:[REDACTED]@")

	// Pattern 4: JWT patterns (eyJ prefix)
	re4 := regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	input = re4.ReplaceAllString(input, "[REDACTED-JWT]")

	return input
}
