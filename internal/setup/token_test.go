package setup

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestGenerateToken_Unique(t *testing.T)        { runTokenCase(t, "unique") }
func TestGenerateToken_256Bits(t *testing.T)       { runTokenCase(t, "bits") }
func TestWriteTokenFile_Atomic(t *testing.T)       { runTokenCase(t, "atomic") }
func TestWriteTokenFile_Perms0600(t *testing.T)    { runTokenCase(t, "perms") }
func TestReadTokenFile_Roundtrips(t *testing.T)    { runTokenCase(t, "read") }
func TestConsumeTokenFile_Removes(t *testing.T)    { runTokenCase(t, "consume") }
func TestConsumeTokenFile_Idempotent(t *testing.T) { runTokenCase(t, "idempotent") }
func TestTokenFile_ConcurrentRace(t *testing.T)    { runTokenCase(t, "race") }

func TestEnsureTokenFile_GeneratesWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), SetupTokenFileName)
	token, err := EnsureTokenFile(path, false)
	if err != nil {
		t.Fatalf("EnsureTokenFile: %v", err)
	}
	if len(token.Raw) != 64 || token.CreatedAt.IsZero() {
		t.Fatalf("invalid generated token: %+v", token)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("token mode=%o want=600", info.Mode().Perm())
	}
}

func TestEnsureTokenFile_PreservesWhenExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), SetupTokenFileName)
	want, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteTokenFile(path, want); err != nil {
		t.Fatal(err)
	}

	got, err := EnsureTokenFile(path, false)
	if err != nil {
		t.Fatalf("EnsureTokenFile: %v", err)
	}
	if got.Raw != want.Raw || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("token changed across restart: got=%+v want=%+v", got, want)
	}
}

func TestEnsureTokenFile_ConcurrentStartup(t *testing.T) {
	path := filepath.Join(t.TempDir(), SetupTokenFileName)
	const starts = 10
	tokens := make([]SetupToken, starts)
	errs := make([]error, starts)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range tokens {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			tokens[i], errs[i] = EnsureTokenFile(path, false)
		}(i)
	}
	close(start)
	wg.Wait()

	for i := range tokens {
		if errs[i] != nil {
			t.Fatalf("startup %d: %v", i, errs[i])
		}
		if tokens[i].Raw != tokens[0].Raw {
			t.Fatalf("startup %d saw token %q; want %q", i, tokens[i].Raw, tokens[0].Raw)
		}
	}
}

func TestEnsureTokenFile_StorageFailure(t *testing.T) {
	parent := t.TempDir()
	blockingFile := filepath.Join(parent, "not-a-directory")
	if err := os.WriteFile(blockingFile, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(blockingFile, SetupTokenFileName)
	if _, err := EnsureTokenFile(path, false); err == nil {
		t.Fatal("expected storage failure")
	}
	entries, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "not-a-directory" {
		t.Fatalf("storage failure left partial files: %v", entries)
	}
}

func TestEnsureTokenFile_ReplayProtection(t *testing.T) {
	path := filepath.Join(t.TempDir(), SetupTokenFileName)
	token, err := EnsureTokenFile(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := ConsumeTokenFile(path); err != nil {
		t.Fatal(err)
	}

	replayed, err := EnsureTokenFile(path, true)
	if err != nil {
		t.Fatalf("configured replay check: %v", err)
	}
	if replayed != (SetupToken{}) {
		t.Fatalf("configured system regenerated consumed token: %+v", replayed)
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("configured system recreated token: %v", err)
	}

	recovered, err := EnsureTokenFile(path, false)
	if err != nil {
		t.Fatalf("empty-system recovery: %v", err)
	}
	if recovered.Raw == "" || recovered.Raw == token.Raw {
		t.Fatalf("empty-system recovery token=%q consumed=%q", recovered.Raw, token.Raw)
	}
}

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
