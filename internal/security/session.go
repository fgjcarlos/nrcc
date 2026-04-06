package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/platform"
)

type SessionManager struct {
	secret []byte
}

func NewSessionManager(secretPath string) (*SessionManager, error) {
	secret, err := ensureSecret(secretPath)
	if err != nil {
		return nil, err
	}

	return &SessionManager{secret: secret}, nil
}

func (sm *SessionManager) Issue(claims model.SessionClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal session claims: %w", err)
	}

	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)
	signature := sm.sign(payloadEnc)
	return payloadEnc + "." + signature, nil
}

func (sm *SessionManager) Verify(token string) (*model.SessionClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid session token format")
	}

	expected := sm.sign(parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, fmt.Errorf("invalid session signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode session payload: %w", err)
	}

	var claims model.SessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("decode session claims: %w", err)
	}

	if claims.Exp <= time.Now().Unix() {
		return nil, fmt.Errorf("session token expired")
	}

	return &claims, nil
}

func (sm *SessionManager) CSRFToken(sessionToken string) string {
	mac := hmac.New(sha256.New, sm.secret)
	_, _ = mac.Write([]byte("csrf:"))
	_, _ = mac.Write([]byte(sessionToken))
	return hex.EncodeToString(mac.Sum(nil))
}

func (sm *SessionManager) VerifyCSRF(sessionToken, csrfToken string) bool {
	expected := sm.CSRFToken(sessionToken)
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(csrfToken)))
}

func (sm *SessionManager) sign(payload string) string {
	mac := hmac.New(sha256.New, sm.secret)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func ensureSecret(secretPath string) ([]byte, error) {
	if platform.Exists(secretPath) {
		raw, err := platform.ReadFile(secretPath)
		if err != nil {
			return nil, err
		}
		secret, err := hex.DecodeString(strings.TrimSpace(string(raw)))
		if err != nil {
			return nil, fmt.Errorf("decode session secret: %w", err)
		}
		return secret, nil
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("generate session secret: %w", err)
	}

	if err := platform.WriteFileIfMissing(secretPath, []byte(hex.EncodeToString(buf)+"\n"), 0o600); err != nil {
		return nil, err
	}

	return buf, nil
}
