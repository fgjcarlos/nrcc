package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Restore swap journal (issue #1024).
//
// A restore publishes the staged tree file by file. Without a journal a
// process death between two renames leaves dataDir holding a mix of the old
// and the new set. The staging directory is therefore also the journal:
//
//   - <staging>/.replaced/<rel> keeps the previous version of every file that
//     already existed, and is only dropped once the whole swap finished.
//   - <staging>/.applied records, per published file, whether the swap
//     replaced an existing file ("keep") or introduced a new one ("new").
//
// RecoverInterruptedRestore walks that journal at boot and puts the previous
// set back, so an interrupted restore always ends as either the full old set
// or the full new set (S1, S2 of #1024).
const (
	restoreStagingPrefix = "restore-staging-"
	restoreReplacedDir   = ".replaced"
	restoreJournalName   = ".applied"

	restoreJournalKeep = "keep"
	restoreJournalNew  = "new"
)

// restoreSwapHook, when non-nil, runs after each published file. Tests use it
// to emulate a process killed in the middle of the rename loop. Production
// leaves it nil.
var restoreSwapHook func(rel string) error

// swapStagingIntoDataDir publishes every staged file listed in the verified
// manifest under dataDir. The set of files moved is derived from the manifest
// (not from a directory listing) so nested paths cannot collide with siblings
// and residue in staging can never be published. Staging lives on the same
// filesystem as dataDir by construction, so os.Rename is atomic per file, and
// the previous version of each file survives in the journal until the whole
// swap completed.
//
// A non-nil error means the journal is still on disk: the caller either rolls
// back now or leaves it for RecoverInterruptedRestore at the next boot.
func (s *BackupService) swapStagingIntoDataDir(stagingDir string, manifest backupMetadata) error {
	if stagingDir == "" {
		return nil
	}
	replacedDir := filepath.Join(stagingDir, restoreReplacedDir)
	if err := os.MkdirAll(replacedDir, 0o750); err != nil {
		return fmt.Errorf("restore journal: %w", err)
	}
	for _, entry := range manifest.Files {
		rel := entry.Path
		if err := validateRestoreRel(rel); err != nil {
			return err
		}
		if err := s.publishRestoreFile(stagingDir, replacedDir, rel); err != nil {
			return err
		}
		if restoreSwapHook != nil {
			if err := restoreSwapHook(rel); err != nil {
				return err
			}
		}
	}
	// Every file is published: the previous set is no longer needed, so
	// dropping the journal is the commit point of the whole restore.
	if err := os.RemoveAll(stagingDir); err != nil {
		return fmt.Errorf("drop restore journal: %w", err)
	}
	return nil
}

func (s *BackupService) publishRestoreFile(stagingDir, replacedDir, rel string) error {
	clean := filepath.FromSlash(rel)
	srcPath := filepath.Join(stagingDir, clean)
	dstPath := filepath.Join(s.dataDir, clean)
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("swap %s: missing in staging: %w", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o750); err != nil {
		return fmt.Errorf("swap %s: mkdir: %w", rel, err)
	}

	kind := restoreJournalNew
	keepPath := filepath.Join(replacedDir, clean)
	_, statErr := os.Stat(dstPath)
	switch {
	case statErr == nil:
		if err := os.MkdirAll(filepath.Dir(keepPath), 0o750); err != nil {
			return fmt.Errorf("swap %s: keep dir: %w", rel, err)
		}
		kind = restoreJournalKeep
	case !errors.Is(statErr, os.ErrNotExist):
		return fmt.Errorf("swap %s: stat: %w", rel, statErr)
	}

	// Journal first: the line is durable before anything in dataDir moves, so
	// no crash can leave a destructive step the rollback cannot undo. The
	// rollback stays correct either way — a "keep" line without a .replaced
	// file means the original never moved (or is already back), and a "new"
	// line only ever authorises removing a file that may not exist yet.
	if err := appendRestoreJournal(stagingDir, rel, kind); err != nil {
		return fmt.Errorf("swap %s: journal: %w", rel, err)
	}

	if kind == restoreJournalKeep {
		if err := os.Rename(dstPath, keepPath); err != nil {
			return fmt.Errorf("swap %s: keep previous: %w", rel, err)
		}
	}
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("swap %s: %w", rel, err)
	}
	return nil
}

// appendRestoreJournal records one published file durably before the swap
// touches dataDir, so a crash can never reach a destructive step the rollback
// cannot undo.
func appendRestoreJournal(stagingDir, rel, kind string) error {
	path := filepath.Join(stagingDir, restoreJournalName)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600) // #nosec G304 -- stagingDir is a service-owned MkdirTemp under dataDir.
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "%s\t%s\n", kind, rel); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

type restoreJournalEntry struct {
	kind string
	rel  string
}

func readRestoreJournal(stagingDir string) ([]restoreJournalEntry, error) {
	path := filepath.Join(stagingDir, restoreJournalName)
	// #nosec G304 -- stagingDir is a service-owned MkdirTemp under dataDir.
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read restore journal: %w", err)
	}
	var entries []restoreJournalEntry
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		kind, rel, ok := strings.Cut(line, "\t")
		if !ok {
			// A torn trailing line means the crash landed while appending;
			// the file it names is still covered by .replaced or by the
			// untouched dataDir copy, so skipping it is safe.
			continue
		}
		if err := validateRestoreRel(rel); err != nil {
			return nil, fmt.Errorf("restore journal: %w", err)
		}
		entries = append(entries, restoreJournalEntry{kind: kind, rel: rel})
	}
	return entries, nil
}

// validateRestoreRel rejects manifest and journal paths that could escape
// dataDir.
func validateRestoreRel(rel string) error {
	if rel == "" || strings.ContainsRune(rel, 0) {
		return fmt.Errorf("manifest entry has invalid path: %q", rel)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if filepath.IsAbs(rel) || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("manifest entry escapes the data dir: %q", rel)
	}
	return nil
}

// RecoverInterruptedRestore rolls back a restore that never finished
// publishing its staged tree (#1024). It returns the staging directory it
// rolled back, or "" when no restore was interrupted.
//
// Call it at boot, before Node-RED starts, next to RecoverInterruptedApply.
func RecoverInterruptedRestore(dataDir string) (string, error) {
	if dataDir == "" {
		return "", nil
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return "", fmt.Errorf("scan data dir for interrupted restore: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), restoreStagingPrefix) {
			continue
		}
		stagingDir := filepath.Join(dataDir, entry.Name())
		if err := rollbackRestoreStaging(dataDir, stagingDir); err != nil {
			// The journal stays so the next boot retries the rollback.
			return "", fmt.Errorf("roll back interrupted restore %s: %w", entry.Name(), err)
		}
		return stagingDir, nil
	}
	return "", nil
}

// rollbackRestoreStaging restores the previous set recorded in stagingDir. The
// journal is the complete record — every line is durable before the swap
// touches dataDir — so retrying a crashed rollback is safe: a keep line whose
// .replaced file is already back finds nothing to do, and a new line only ever
// removes the file the swap introduced.
func rollbackRestoreStaging(dataDir, stagingDir string) error {
	entries, err := readRestoreJournal(stagingDir)
	if err != nil {
		return err
	}
	replacedDir := filepath.Join(stagingDir, restoreReplacedDir)
	for _, entry := range entries {
		rel := filepath.FromSlash(entry.rel)
		switch entry.kind {
		case restoreJournalKeep:
			keepPath := filepath.Join(replacedDir, rel)
			if _, err := os.Stat(keepPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return err
			}
			dstPath := filepath.Join(dataDir, rel)
			if err := os.MkdirAll(filepath.Dir(dstPath), 0o750); err != nil {
				return err
			}
			if err := os.Rename(keepPath, dstPath); err != nil {
				return err
			}
		case restoreJournalNew:
			if err := os.Remove(filepath.Join(dataDir, rel)); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	return os.RemoveAll(stagingDir)
}

// manifestRestoresFile reports whether the archive replaces rel, which decides
// whether a restore has to reinstall node_modules (issue #1024).
func manifestRestoresFile(manifest backupMetadata, rel string) bool {
	for _, entry := range manifest.Files {
		if entry.Path == rel {
			return true
		}
	}
	return false
}
