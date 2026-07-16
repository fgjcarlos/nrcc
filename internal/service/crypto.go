package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

const encPrefix = "enc:"

// GenerateJWTSecret returns 32 random bytes hex-encoded. Moved here from
// the removed installer package; only main.go's JWT bootstrap uses it.
func GenerateJWTSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	return fmt.Sprintf("%x", b), nil
}

func deriveKey(passphrase string) []byte {
	hash := sha256.Sum256([]byte(passphrase))
	return hash[:]
}

func Encrypt(plaintext, key string) (string, error) {
	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encoded, key string) (string, error) {
	if !strings.HasPrefix(encoded, encPrefix) {
		return encoded, nil
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, encPrefix))
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	block, err := aes.NewCipher(deriveKey(key))
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptForDownload wraps raw backup bytes with AES-256-GCM using the
// operator's passphrase and returns the enc:-prefixed envelope. The
// handler uses this to compute a Content-Length *before* writing any
// response body, so a partial-write error becomes a 500 instead of an
// HTTP 200 with truncated ciphertext.
//
// ponytail: still buffers the full encrypted blob in memory; cap is the
// same as maxBackupEntrySize (200 MiB) plus GCM+base64 overhead. If
// archives grow past hundreds of MiB, switch to a streaming
// cipher.NewGCM(NewEncrypter) over io.Pipe.
func EncryptForDownload(zipBytes []byte, password string) ([]byte, error) {
	s, err := EncryptBytes(zipBytes, password)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// EncryptBytes wraps Encrypt for byte payloads, returning the enc:-prefixed
// base64 ciphertext. Kept as the canonical byte-oriented primitive; callers
// that need it for non-download purposes (e.g. tests) use this directly.
func EncryptBytes(plaintext []byte, key string) (string, error) {
	return Encrypt(string(plaintext), key)
}

// DecryptBytes is the symmetric counterpart of EncryptBytes.
func DecryptBytes(encoded, key string) ([]byte, error) {
	s, err := Decrypt(encoded, key)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// IsEncrypted reports whether value carries the enc: ciphertext prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encPrefix)
}
