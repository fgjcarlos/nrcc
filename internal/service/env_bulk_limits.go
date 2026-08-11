package service

import (
	"fmt"
	"os"
	"strconv"
)

const defaultBulkMaxEntries = 1000
const defaultBulkMaxEntryBytes = 8192

type BulkLimits struct {
	MaxEntries    int
	MaxEntryBytes int
}

func DefaultBulkLimits() BulkLimits {
	return BulkLimits{MaxEntries: defaultBulkMaxEntries, MaxEntryBytes: defaultBulkMaxEntryBytes}
}

func (l BulkLimits) resolved() (BulkLimits, error) {
	if l.MaxEntries < 0 || l.MaxEntryBytes < 0 {
		return BulkLimits{}, fmt.Errorf("bulk limits cannot be negative")
	}
	defaults := DefaultBulkLimits()
	if l.MaxEntries == 0 {
		l.MaxEntries = defaults.MaxEntries
	}
	if l.MaxEntryBytes == 0 {
		l.MaxEntryBytes = defaults.MaxEntryBytes
	}
	return l, nil
}

func BulkLimitsFromEnv() (BulkLimits, error) {
	maxEntries, err := parseBulkLimit("NRCC_BULK_MAX_ENTRIES", os.Getenv("NRCC_BULK_MAX_ENTRIES"))
	if err != nil {
		return BulkLimits{}, err
	}
	maxEntryBytes, err := parseBulkLimit("NRCC_BULK_MAX_ENTRY_BYTES", os.Getenv("NRCC_BULK_MAX_ENTRY_BYTES"))
	if err != nil {
		return BulkLimits{}, err
	}
	return (BulkLimits{MaxEntries: maxEntries, MaxEntryBytes: maxEntryBytes}).resolved()
}

func parseBulkLimit(name, raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}
	return value, nil
}
