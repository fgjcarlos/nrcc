package service

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestValidateSecret(t *testing.T) {
	placeholders := []string{
		"cc-secret-change-in-production",
		"change-me-in-production",
		"dev-secret-not-for-production",
	}

	// Empty value is "not provided" — not a rejection case. Callers
	// that require a value must enforce that separately.
	if err := ValidateSecret("JWT_SECRET", ""); err != nil {
		t.Fatalf("empty value must not be rejected: %v", err)
	}

	// Each placeholder must be rejected regardless of casing, and
	// the error must mention the env var name so the operator knows
	// where to fix it.
	for _, p := range placeholders {
		t.Run(p, func(t *testing.T) {
			err := ValidateSecret("NRCC_ENCRYPTION_KEY", p)
			if err == nil {
				t.Fatalf("expected rejection of %q", p)
			}
			if !strings.Contains(err.Error(), "NRCC_ENCRYPTION_KEY") {
				t.Errorf("error must mention the env var name, got: %v", err)
			}
		})

		t.Run(strings.ToUpper(p), func(t *testing.T) {
			err := ValidateSecret("NRCC_ENCRYPTION_KEY", strings.ToUpper(p))
			if err == nil {
				t.Fatalf("expected rejection of upper-cased %q", p)
			}
		})
	}

	// Real-looking values must pass through unchanged.
	for _, v := range []string{
		"my-real-production-secret",
		"open sesame: a long passphrase with spaces",
		strings.Repeat("a", 64),
	} {
		if err := ValidateSecret("JWT_SECRET", v); err != nil {
			t.Errorf("real secret %q must not be rejected: %v", v, err)
		}
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := "my-secret-encryption-key"
	values := []string{
		"simple-password",
		"",
		"p@$$w0rd!#&*(){}",
		"multi\nline\nvalue",
		strings.Repeat("a", 10000),
	}

	for _, plaintext := range values {
		encrypted, err := Encrypt(plaintext, key)
		if err != nil {
			t.Fatalf("Encrypt(%q) error: %v", plaintext, err)
		}

		if !strings.HasPrefix(encrypted, encPrefix) {
			t.Errorf("encrypted value should have %q prefix, got %q", encPrefix, encrypted[:10])
		}

		decrypted, err := Decrypt(encrypted, key)
		if err != nil {
			t.Fatalf("Decrypt() error: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("roundtrip failed: got %q, want %q", decrypted, plaintext)
		}
	}
}

func TestEncrypt_DifferentNonces(t *testing.T) {
	key := "test-key"
	plaintext := "same-value"

	a, _ := Encrypt(plaintext, key)
	b, _ := Encrypt(plaintext, key)

	if a == b {
		t.Error("two encryptions of the same value should produce different ciphertexts")
	}
}

func TestDecrypt_PlaintextPassthrough(t *testing.T) {
	result, err := Decrypt("not-encrypted-value", "any-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "not-encrypted-value" {
		t.Errorf("plaintext passthrough failed: got %q", result)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	encrypted, _ := Encrypt("secret", "correct-key")

	_, err := Decrypt(encrypted, "wrong-key")
	if err == nil {
		t.Error("decryption with wrong key should fail")
	}
}

func TestDecrypt_CorruptedData(t *testing.T) {
	_, err := Decrypt(encPrefix+"not-valid-base64!!!", "key")
	if err == nil {
		t.Error("corrupted base64 should fail")
	}
}

func TestIsEncrypted(t *testing.T) {
	if IsEncrypted("plaintext") {
		t.Error("plaintext should not be detected as encrypted")
	}
	if !IsEncrypted(encPrefix + "something") {
		t.Error("prefixed value should be detected as encrypted")
	}
}

// TestEncryptStreamRoundTrip covers the streaming AEAD round-trip over a
// payload larger than streamingChunk so the loop crosses at least one
// chunk boundary. Asserts the envelope is small relative to the input
// (no whole-blob buffering) and that wrong passwords fail at the GCM
// tag check.
func TestEncryptStreamRoundTrip(t *testing.T) {
	payload := bytes.Repeat([]byte("ABCDEFGH"), 512*1024) // 4 MiB
	var enc bytes.Buffer
	if err := EncryptStream(bytes.NewReader(payload), "stream-pw", &enc); err != nil {
		t.Fatalf("EncryptStream: %v", err)
	}
	// Header (7 magic + 16 salt + 12 nonce = 35) + at least one chunk
	// (4 len + 64 KiB plaintext + 16 tag = 65620) + 4-byte EOF marker.
	if enc.Len() < 32+4+16 {
		t.Fatalf("encrypted envelope suspiciously small: %d bytes", enc.Len())
	}

	var dec bytes.Buffer
	if err := DecryptStream(&enc, "stream-pw", &dec); err != nil {
		t.Fatalf("DecryptStream: %v", err)
	}
	if !bytes.Equal(dec.Bytes(), payload) {
		t.Fatalf("round-trip mismatch: got %d bytes, want %d", dec.Len(), len(payload))
	}

	var wrong bytes.Buffer
	if err := DecryptStream(bytes.NewReader(enc.Bytes()), "wrong-pw", &wrong); err == nil {
		t.Fatal("DecryptStream with wrong password must fail")
	}
}

// TestEncryptStreamHeaderMagic rejects inputs that do not start with the
// enc:v2: magic, so an operator who accidentally feeds a legacy enc:
// envelope into the streaming path gets a clear error.
func TestEncryptStreamHeaderMagic(t *testing.T) {
	if err := DecryptStream(bytes.NewReader([]byte("enc:abcdef==")), "any", io.Discard); err == nil {
		t.Fatal("DecryptStream must reject non-streaming input")
	}
}

// TestV2EnvelopeIsNotLegacy is a structural pin against silent
// regressions: EncryptStream must NOT produce an `enc:base64(...)`
// envelope, and Encrypt must NOT produce the binary `enc:v2:` header.
// If a future refactor unifies the two envelopes under the SHA-256
// derivation (or under argon2id without the magic), one of these
// round-trips breaks loudly.
func TestV2EnvelopeIsNotLegacy(t *testing.T) {
	var stream bytes.Buffer
	if err := EncryptStream(bytes.NewReader([]byte("a-legacy-string-of-modest-length")), "pin", &stream); err != nil {
		t.Fatalf("EncryptStream: %v", err)
	}
	if !bytes.HasPrefix(stream.Bytes(), []byte(encV2Magic)) {
		t.Fatalf("EncryptStream envelope must start with %q, got %q", encV2Magic, stream.Bytes()[:7])
	}

	legacy, err := EncryptBytes([]byte("a-legacy-string-of-modest-length"), "pin")
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}
	if !strings.HasPrefix(legacy, encPrefix) || strings.HasPrefix(legacy, encV2Magic) {
		t.Fatalf("EncryptBytes must produce legacy enc: envelope, got %q", legacy[:min(8, len(legacy))])
	}
}
