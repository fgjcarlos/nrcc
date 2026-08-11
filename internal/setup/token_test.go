package setup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestGenerateToken_Unique(t *testing.T)                  { runTokenCase(t, "unique") }
func TestGenerateToken_256Bits(t *testing.T)                 { runTokenCase(t, "bits") }
func TestWriteTokenFile_Atomic(t *testing.T)                 { runTokenCase(t, "atomic") }
func TestWriteTokenFile_Perms0600(t *testing.T)              { runTokenCase(t, "perms") }
func TestReadTokenFile_Roundtrips(t *testing.T)              { runTokenCase(t, "read") }
func TestConsumeTokenFile_Removes(t *testing.T)              { runTokenCase(t, "consume") }
func TestConsumeTokenFile_Idempotent(t *testing.T)           { runTokenCase(t, "idempotent") }
func TestTokenFile_ConcurrentRace(t *testing.T)              { runTokenCase(t, "race") }

func runTokenCase(t *testing.T, scenario string) {
	t.Helper()
	dir, path := t.TempDir(), filepath.Join(t.TempDir(), SetupTokenFileName)
	makeToken := func() SetupToken {
		token, err := GenerateToken()
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	write := func(token SetupToken) {
		if err := WriteTokenFile(path, token); err != nil {
			t.Fatal(err)
		}
	}
	switch scenario {
	case "unique":
		seen := map[string]bool{}
		for i := 0; i < 1000; i++ {
			token := makeToken()
			if seen[token.Raw] {
				t.Fatal("duplicate token")
			}
			seen[token.Raw] = true
		}
	case "bits":
		token := makeToken()
		if len(token.Raw) != 64 || token.CreatedAt.IsZero() {
			t.Fatalf("invalid token: %+v", token)
		}
	case "atomic":
		path = filepath.Join(dir, SetupTokenFileName)
		write(makeToken())
		entries, _ := os.ReadDir(dir)
		if len(entries) != 1 || entries[0].Name() != SetupTokenFileName {
			t.Fatalf("unexpected files: %v", entries)
		}
	case "perms":
		write(makeToken())
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("mode=%v err=%v", info.Mode(), err)
		}
	case "read":
		want := makeToken()
		write(want)
		got, err := ReadTokenFile(path)
		if err != nil || got.Raw != want.Raw || !got.CreatedAt.Equal(want.CreatedAt) {
			t.Fatalf("got=%+v want=%+v err=%v", got, want, err)
		}
	case "consume", "idempotent":
		if scenario == "consume" {
			write(makeToken())
		}
		if err := ConsumeTokenFile(path); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("stat error=%v", err)
		}
	case "race":
		tokens := make([]SetupToken, 10)
		for i := range tokens {
			tokens[i] = makeToken()
		}
		var wg sync.WaitGroup
		for _, token := range tokens {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := WriteTokenFile(path, token); err != nil {
					t.Error(err)
				}
			}()
		}
		wg.Wait()
		if token, err := ReadTokenFile(path); err != nil || len(token.Raw) != 64 {
			t.Fatalf("token=%+v err=%v", token, err)
		}
	}
}
