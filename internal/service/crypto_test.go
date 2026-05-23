package service

import (
	"strings"
	"testing"
)

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
