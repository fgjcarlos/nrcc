package service

import "errors"

// ErrDataDirBusy is returned when another destructive data-dir operation
// (restore, backup creation or apply) is already running. Use errors.Is to
// detect it.
//
// Issue #1024: one data directory runs one such operation at a time, so a
// second restore, a scheduled backup or an apply can never observe or write a
// half-restored tree.
var ErrDataDirBusy = errors.New("another restore, backup or apply is in progress for this data directory")
