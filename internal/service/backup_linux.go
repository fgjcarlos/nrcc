//go:build linux

package service

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// OpenForDownload validates the id, confirms the backup exists and opens it,
// returning a reader and the file size. Callers (HTTP handlers) can use this to
// detect missing/unreadable backups and set Content-Length BEFORE writing any
// response body, avoiding truncated downloads served with a 200 status.
//
// The open uses O_NOFOLLOW so a symlink substituted into the backup path
// between ValidateBackupID and open returns ELOOP rather than reading the
// symlink target. openNoFollow additionally fstats the fd and rejects any
// symlink that survives the kernel open — defense-in-depth.
func (s *BackupService) OpenForDownload(id string) (io.ReadCloser, int64, error) {
	if err := ValidateBackupID(id); err != nil {
		return nil, 0, err
	}
	backupPath := filepath.Join(s.backupDir, id+".zip")

	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, 0, fmt.Errorf("backup not found: %w", err)
	}

	// #nosec G304 -- backupPath is built from operator-supplied dataDir + a constant filename; not request-derived.
	file, err := openNoFollow(backupPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open backup: %w", err)
	}
	return file, info.Size(), nil
}

// Download streams a backup file to the response writer. If password is
// non-empty the zip bytes are wrapped with AES-256-GCM (see Encrypt); the
// client decrypts with the same passphrase. This lets an operator transfer a
// backup containing credentials/secrets off-host without exposing them in the
// raw archive.
//
// The encrypted path streams in 64 KiB chunks via EncryptStream so peak
// memory is bounded well below the archive size; the handler must set
// headers (including Transfer-Encoding: chunked when no Content-Length is
// available) BEFORE calling Download.
func (s *BackupService) Download(id string, w io.Writer, password string) error {
	rc, _, err := s.OpenForDownload(id)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	if password == "" {
		_, err := io.Copy(w, rc)
		if err != nil {
			return fmt.Errorf("failed to download backup: %w", err)
		}
		return nil
	}

	if err := EncryptStream(rc, password, w); err != nil {
		return fmt.Errorf("failed to stream encrypted backup: %w", err)
	}
	return nil
}