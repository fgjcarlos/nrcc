package service

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// signClaims signs an arbitrary claims map with the same secret the test auth
// service uses, producing a structurally valid token with attacker/edge-case
// claim shapes.
func signClaims(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

// TestVerifyToken_MalformedClaimsReturnError is the #277 regression: a token
// with missing or wrong-typed claims must return an error, never panic the
// request via a bare type assertion.
func TestVerifyToken_MalformedClaimsReturnError(t *testing.T) {
	svc := newTestAuthService(t)

	cases := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{"userId is a number", jwt.MapClaims{"userId": 123, "username": "a", "role": "admin", "exp": 9999999999.0, "iat": 1.0}},
		{"role missing", jwt.MapClaims{"userId": "u1", "username": "a", "exp": 9999999999.0, "iat": 1.0}},
		{"exp is a string", jwt.MapClaims{"userId": "u1", "username": "a", "role": "admin", "exp": "soon", "iat": 1.0}},
		{"username missing", jwt.MapClaims{"userId": "u1", "role": "admin", "exp": 9999999999.0, "iat": 1.0}},
		{"all claims absent", jwt.MapClaims{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// A panic here (bare type assertion) fails the test instead of
			// crashing the whole run.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("VerifyToken panicked on malformed claims: %v", r)
				}
			}()

			tokenStr := signClaims(t, tc.claims)
			claims, err := svc.VerifyToken(tokenStr)
			if err == nil {
				t.Fatalf("expected error for malformed claims, got claims=%+v", claims)
			}
		})
	}
}

// TestVerifyToken_ValidClaimsStillWork guards against over-tightening: a normal
// token must still verify and map every field correctly.
func TestVerifyToken_ValidClaimsStillWork(t *testing.T) {
	svc := newTestAuthService(t)
	tokenStr := signClaims(t, jwt.MapClaims{
		"userId":   "u1",
		"username": "admin",
		"role":     "admin",
		"exp":      9999999999.0,
		"iat":      1700000000.0,
	})

	claims, err := svc.VerifyToken(tokenStr)
	if err != nil {
		t.Fatalf("VerifyToken on valid claims: %v", err)
	}
	if claims.UserID != "u1" || claims.Username != "admin" || string(claims.Role) != "admin" {
		t.Fatalf("claims not mapped correctly: %+v", claims)
	}
	if claims.ExpiresAt != 9999999999 || claims.IssuedAt != 1700000000 {
		t.Fatalf("numeric claims not mapped correctly: %+v", claims)
	}
}
