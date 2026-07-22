package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	encPrefix  = "enc:"
	encV2Magic = "enc:v2:" // header prefix identifying the streaming envelope
	encV2Salt  = 16
	encV2Nonce = 12
	// streamingChunk is the plaintext chunk size for EncryptStream. 64 KiB
	// keeps memory bounded well below the prior whole-zip read while
	// staying large enough that GCM overhead per chunk stays negligible.
	streamingChunk = 64 * 1024

	// argon2id parameters for v2 envelope key derivation. Fixed across
	// deployments — see #478 for the rationale. mem=64 MiB, t=3, p=2 gives
	// ~250 ms per derivation on the reference host, which is acceptable
	// for an operator-only download path.
	argon2Memory = 64 * 1024
	argon2Time   = 3
	argon2Threads = 2
)

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

// deriveKeyArgon2id derives a 32-byte AES key with argon2id using the
// given salt. Used by the v2 streaming envelope; the legacy enc: path
// still uses single SHA-256 because its input is operator-readable
// config (not a high-value secret) and migration would break existing
// stored values.
func deriveKeyArgon2id(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argon2Time, argon2Memory, argon2Threads, 32)
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

// EncryptStream wraps rc in a streaming AES-256-GCM envelope and writes
// it to w. Peak memory is bounded by streamingChunk (64 KiB plaintext)
// plus one AEAD ciphertext chunk, regardless of the total payload size.
//
// Envelope layout (binary, no base64):
//   - "enc:v2:"             7 bytes  ASCII magic + version
//   - salt                  16 bytes random per call
//   - base nonce            12 bytes random per call (the per-chunk nonce is
//                            XORed with the chunk counter in the low 8 bytes)
//   - chunks                repeat until EOF:
//       chunk_len           4 bytes  BE, ciphertext length (incl. 16B tag)
//                                  zero marks end-of-stream
//       chunk_nonce         12 bytes derived from base + counter
//       chunk_ciphertext    chunk_len bytes
//
// The handler is responsible for setting headers (Transfer-Encoding:
// chunked, no Content-Length) BEFORE invoking this function; the
// returned error after any bytes have hit the wire is a peer-truncation
// case the same way the legacy path was.
//
// ponytail: GCM-with-counter nonces are a deliberate deviation from the
// canonical chunked-AEAD (RFC 9580 §5.4). It is correct for an
// operator-only download where the only attacker who can tamper with a
// chunk also controls the whole stream; upgrade if a partial-trust
// scenario (e.g. browser-decrypted streaming) ever appears.
func EncryptStream(rc io.Reader, password string, w io.Writer) error {
	salt := make([]byte, encV2Salt)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	key := deriveKeyArgon2id(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}

	// Header carries the salt so the recipient can derive the same key.
	header := make([]byte, 0, len(encV2Magic)+encV2Salt)
	header = append(header, []byte(encV2Magic)...)
	header = append(header, salt...)
	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// One fresh nonce per chunk; low 8 bytes are the big-endian counter
	// so chunks can be reordered safely under the same password.
	baseNonce := make([]byte, encV2Nonce)
	if _, err := rand.Read(baseNonce); err != nil {
		return fmt.Errorf("generate nonce: %w", err)
	}
	if _, err := w.Write(baseNonce); err != nil {
		return fmt.Errorf("write nonce: %w", err)
	}

	buf := make([]byte, streamingChunk)
	var counter uint64
	for {
		n, err := io.ReadFull(rc, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("read plaintext: %w", err)
		}
		if n > 0 {
			chunkNonce := deriveChunkNonce(baseNonce, counter)
			sealed := gcm.Seal(nil, chunkNonce, buf[:n], nil)
			lenBuf := make([]byte, 4)
			binary.BigEndian.PutUint32(lenBuf, uint32(len(sealed)))
			if _, err := w.Write(lenBuf); err != nil {
				return fmt.Errorf("write chunk len: %w", err)
			}
			if _, err := w.Write(sealed); err != nil {
				return fmt.Errorf("write chunk: %w", err)
			}
			counter++
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
	}

	// Zero-length final frame marks end-of-stream.
	end := make([]byte, 4)
	if _, err := w.Write(end); err != nil {
		return fmt.Errorf("write EOF marker: %w", err)
	}
	return nil
}

// deriveChunkNonce returns a copy of baseNonce with the low 8 bytes
// replaced by the big-endian counter. Counter stays in its own 8-byte
// region so chunks are distinguishable even if the high 4 bytes happen
// to collide with another chunk's high 4 bytes.
func deriveChunkNonce(base []byte, counter uint64) []byte {
	out := make([]byte, len(base))
	copy(out, base)
	binary.BigEndian.PutUint64(out[4:], counter)
	return out
}

// DecryptStream is the symmetric counterpart of EncryptStream: it reads
// the streaming envelope from r and writes the recovered plaintext to w.
// Returns an error if the envelope is malformed or any chunk fails its
// AEAD tag check.
func DecryptStream(r io.Reader, password string, w io.Writer) error {
	header := make([]byte, len(encV2Magic)+encV2Salt+encV2Nonce)
	if _, err := io.ReadFull(r, header); err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	if string(header[:len(encV2Magic)]) != encV2Magic {
		return fmt.Errorf("not a streaming envelope")
	}
	salt := header[len(encV2Magic) : len(encV2Magic)+encV2Salt]
	baseNonce := header[len(encV2Magic)+encV2Salt:]

	key := deriveKeyArgon2id(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}

	lenBuf := make([]byte, 4)
	var counter uint64
	for {
		if _, err := io.ReadFull(r, lenBuf); err != nil {
			return fmt.Errorf("read chunk len: %w", err)
		}
		chunkLen := binary.BigEndian.Uint32(lenBuf)
		if chunkLen == 0 {
			return nil
		}
		sealed := make([]byte, chunkLen)
		if _, err := io.ReadFull(r, sealed); err != nil {
			return fmt.Errorf("read chunk: %w", err)
		}
		chunkNonce := deriveChunkNonce(baseNonce, counter)
		plain, err := gcm.Open(nil, chunkNonce, sealed, nil)
		if err != nil {
			return fmt.Errorf("decrypt chunk %d: %w", counter, err)
		}
		if _, err := w.Write(plain); err != nil {
			return fmt.Errorf("write plaintext: %w", err)
		}
		counter++
	}
}
