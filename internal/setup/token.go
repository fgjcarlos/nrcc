package setup

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const SetupTokenFileName = ".nrcc-setup-token"

type SetupToken struct {
	Raw       string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
}

func GenerateToken() (SetupToken, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return SetupToken{}, err
	}
	return SetupToken{Raw: hex.EncodeToString(raw), CreatedAt: time.Now().UTC()}, nil
}

func ReadTokenFile(path string) (SetupToken, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SetupToken{}, err
	}
	var token SetupToken
	if err := json.Unmarshal(data, &token); err != nil {
		return SetupToken{}, err
	}
	if err := validateToken(token); err != nil {
		return SetupToken{}, err
	}
	return token, nil
}

func WriteTokenFile(path string, token SetupToken) error {
	if err := validateToken(token); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".nrcc-setup-token-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := json.NewEncoder(tmp).Encode(token); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func ConsumeTokenFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func EnsureTokenFile(path string, configured bool) error {
	if _, err := ReadTokenFile(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if configured {
		return nil
	}
	token, err := GenerateToken()
	if err != nil {
		return err
	}
	return WriteTokenFile(path, token)
}

func (t SetupToken) String() string { return t.Raw }

func validateToken(token SetupToken) error {
	decoded, err := hex.DecodeString(token.Raw)
	if err != nil || len(decoded) != 32 {
		return errors.New("setup token must be 256-bit hexadecimal")
	}
	if token.CreatedAt.IsZero() {
		return errors.New("setup token creation time is required")
	}
	return nil
}
