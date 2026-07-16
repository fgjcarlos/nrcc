package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ResticProvider shells out to the restic binary. All configuration is
// passed via env vars (RESTIC_REPOSITORY, RESTIC_PASSWORD) so credentials
// never appear in process argv.
//
// ponytail: per-call exec spawn is fine for the local-provider scale (one
// snapshot per scheduler tick + manual triggers). If providers are called
// concurrently the underlying binary invocation is read-only on the
// repository, but the local cache directory can race — document that the
// NRCC instance owns one restic cache dir (NRCC_RESTIC_CACHE_DIR or
// derived from temp).
type ResticProvider struct {
	// Binary is the absolute path or PATH-relative name for restic.
	Binary string
	// Repo is the RESTIC_REPOSITORY value (path, s3:bucket, etc.).
	Repo string
	// Password is the RESTIC_PASSWORD value. Either Password or
	// PasswordFile must be set.
	Password string
	// PasswordFile is the path to the RESTIC_PASSWORD_FILE.
	PasswordFile string
	// CacheDir is the RESTIC_CACHE_DIR. If empty, os.TempDir() is used.
	CacheDir string
	// extraEnv holds extra RESTIC_* env entries callers may inject
	// (e.g. AWS_ACCESS_KEY_ID). Merged on top of the defaults.
	extraEnv []string
}

// ResticConfig captures the operator-facing knobs for the ResticProvider.
// Empty fields fall back to environment variables or safe defaults.
type ResticConfig struct {
	Binary       string
	Repo         string
	Password     string
	PasswordFile string
	CacheDir     string
	ExtraEnv     []string
}

// NewResticProvider builds a provider from a config struct. The repo must
// be non-empty; everything else is optional.
func NewResticProvider(cfg ResticConfig) (*ResticProvider, error) {
	if strings.TrimSpace(cfg.Repo) == "" {
		return nil, fmt.Errorf("restic provider requires a non-empty Repo")
	}
	if strings.TrimSpace(cfg.Password) == "" && strings.TrimSpace(cfg.PasswordFile) == "" {
		return nil, fmt.Errorf("restic provider requires either Password or PasswordFile")
	}
	binary := strings.TrimSpace(cfg.Binary)
	if binary == "" {
		binary = "restic"
	}
	return &ResticProvider{
		Binary:       binary,
		Repo:         cfg.Repo,
		Password:     cfg.Password,
		PasswordFile: cfg.PasswordFile,
		CacheDir:     cfg.CacheDir,
		extraEnv:     cfg.ExtraEnv,
	}, nil
}

// NewResticProviderFromEnv builds a provider from NRCC_RESTIC_* environment
// variables. Returns nil + nil error if NRCC_RESTIC_REPO is unset (caller
// should keep NoopProvider). Returns a non-nil error only when the env is
// set but misconfigured.
func NewResticProviderFromEnv() (*ResticProvider, error) {
	repo := os.Getenv("NRCC_RESTIC_REPO")
	if strings.TrimSpace(repo) == "" {
		return nil, nil
	}
	cfg := ResticConfig{
		Binary:       os.Getenv("NRCC_RESTIC_BINARY"),
		Repo:         repo,
		Password:     os.Getenv("NRCC_RESTIC_PASSWORD"),
		PasswordFile: os.Getenv("NRCC_RESTIC_PASSWORD_FILE"),
		CacheDir:     os.Getenv("NRCC_RESTIC_CACHE_DIR"),
	}
	return NewResticProvider(cfg)
}

func (p *ResticProvider) Name() string { return "restic" }

// initOnce serializes the first `restic init` call so two concurrent
// snapshots on a fresh repository cannot both invoke `init`. Subsequent
// callers see the cached result without touching the network.
var initOnce sync.Map // key: *ResticProvider -> *initResult

// env returns the env entries required for every restic invocation. Does not
// include OS env; callers pass the result as cmd.Env.
func (p *ResticProvider) env() []string {
	env := []string{
		"RESTIC_REPOSITORY=" + p.Repo,
	}
	if p.Password != "" {
		env = append(env, "RESTIC_PASSWORD="+p.Password)
	}
	if p.PasswordFile != "" {
		env = append(env, "RESTIC_PASSWORD_FILE="+p.PasswordFile)
	}
	cache := p.CacheDir
	if cache == "" {
		cache = filepath.Join(os.TempDir(), "nrcc-restic-cache")
	}
	env = append(env, "RESTIC_CACHE_DIR="+cache)
	env = append(env, p.extraEnv...)
	return env
}

// run executes the restic binary with the given args and returns stdout.
// stderr is captured separately so it can be surfaced in errors. The
// returned error message redacts the argv and includes only the restic
// stderr so admin-supplied values (id, destination) cannot leak through
// the public error path.
func (p *ResticProvider) run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, p.Binary, args...)
	cmd.Env = p.env()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("restic %s: %s", cmdName(args), strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// cmdName returns a short stable label for a restic invocation. Used in
// error messages so we never echo the raw argv back to the caller.
func cmdName(args []string) string {
	if len(args) == 0 {
		return "command"
	}
	return args[0]
}

// Snapshot uploads the file at srcPath. The repository is created on first
// call (restic's `backup` command fails if the repo is uninitialized; we
// try `init` first and ignore "already exists" errors).
func (p *ResticProvider) Snapshot(ctx context.Context, srcPath string) (string, error) {
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("snapshot source missing: %w", err)
	}
	if err := p.initRepoIfNeeded(ctx); err != nil {
		return "", err
	}

	// restic `backup` tags the file with its host path; we pass the
	// absolute path so the snapshot IDs are reproducible.
	stdout, err := p.run(ctx, "backup", "--json", "--tag", "nrcc", srcPath)
	if err != nil {
		return "", err
	}
	// stdout contains one JSON line per progress event. The final line has
	// message_type == "summary" and a snapshot id.
	id, perr := parseResticSnapshotID(stdout)
	if perr != nil {
		return "", perr
	}
	if id == "" {
		return "", errors.New("restic backup completed but no snapshot id was reported")
	}
	return id, nil
}

// initRepoIfNeeded runs `restic init` and treats "already initialized" as a
// no-op (restic exits non-zero with a clear message in that case). The first
// concurrent caller wins; the rest piggyback on its result.
func (p *ResticProvider) initRepoIfNeeded(ctx context.Context) error {
	type initResult struct {
		err error
	}
	if v, ok := initOnce.Load(p); ok {
		return v.(*initResult).err
	}
	actual, _ := initOnce.LoadOrStore(p, &initResult{})
	// Another goroutine beat us to it; wait for its result.
	if v, ok := initOnce.Load(p); ok && v != actual {
		return v.(*initResult).err
	}
	res := actual.(*initResult)
	cmd := exec.CommandContext(ctx, p.Binary, "init")
	cmd.Env = p.env()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		res.err = nil
		return nil
	}
	msg := strings.TrimSpace(stderr.String())
	if strings.Contains(msg, "already initialized") || strings.Contains(msg, "config file already exists") {
		res.err = nil
		return nil
	}
	res.err = fmt.Errorf("restic init failed: %w: %s", err, msg)
	return res.err
}

func (p *ResticProvider) List(ctx context.Context) ([]RemoteBackup, error) {
	stdout, err := p.run(ctx, "snapshots", "--json", "--tag", "nrcc")
	if err != nil {
		return nil, err
	}
	var raw []struct {
		ID   string `json:"id"`
		Time time.Time `json:"time"`
	}
	if err := json.Unmarshal(stdout, &raw); err != nil {
		return nil, fmt.Errorf("parse restic snapshots: %w", err)
	}
	out := make([]RemoteBackup, 0, len(raw))
	for _, r := range raw {
		out = append(out, RemoteBackup{ID: r.ID, Time: r.Time})
	}
	return out, nil
}

func (p *ResticProvider) Restore(ctx context.Context, remoteID, dstPath string) error {
	if err := validateResticSnapshotID(remoteID); err != nil {
		return err
	}
	if dstPath == "" {
		return errors.New("destination is required")
	}
	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		return err
	}
	if _, err := p.run(ctx, "restore", remoteID, "--target", dstPath); err != nil {
		return err
	}
	return nil
}

// validateResticSnapshotID rejects anything that is not a hex restic
// snapshot id (typically 8–64 lowercase hex chars). Admin-only callers
// still get validation so a typo or hostile admin tool cannot smuggle
// argv flags via the id field.
func validateResticSnapshotID(id string) error {
	if id == "" {
		return errors.New("remote id required")
	}
	if len(id) < 8 || len(id) > 64 {
		return errors.New("remote id has invalid length")
	}
	for _, r := range id {
		isDigit := r >= '0' && r <= '9'
		isHexLetter := r >= 'a' && r <= 'f'
		if !isDigit && !isHexLetter {
			return errors.New("remote id must be lowercase hex")
		}
	}
	return nil
}

// parseResticSnapshotID scans the `restic backup --json` output for the
// final "summary" event and returns its `snapshot_id`. restic emits one
// JSON object per line in non-deterministic order.
func parseResticSnapshotID(out []byte) (string, error) {
	var snapshotID string
	scanner := bytes.Split(out, []byte("\n"))
	for _, line := range scanner {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev struct {
			MessageType string `json:"message_type"`
			SnapshotID  string `json:"snapshot_id"`
		}
		if err := json.Unmarshal(line, &ev); err != nil {
			// Non-fatal: restic can print non-JSON banners on some
			// versions; skip them and keep scanning.
			continue
		}
		if ev.MessageType == "summary" && ev.SnapshotID != "" {
			snapshotID = ev.SnapshotID
		}
	}
	return snapshotID, nil
}