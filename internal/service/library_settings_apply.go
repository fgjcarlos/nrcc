package service

import (
	"context"
	"errors"
	"path/filepath"
)

// LibrarySettingsUpdater returns the LibraryService settings hook (#1016).
//
// While running reports a managed Node-RED process, the transform goes
// through the apply coordinator: snapshot, atomic write, restart, readiness
// and rollback. The hook then reports restarted=true so the library service
// does not restart Node-RED a second time.
//
// A stopped runtime stays stopped, and an environment without a restart
// adapter keeps working: both fall back to the validated atomic write of
// ConfigService.TransformRawSettings and report restarted=false.
func LibrarySettingsUpdater(coordinator *ApplyCoordinator, config *ConfigService, running func() bool) func(context.Context, func(string) string) (bool, error) {
	return func(ctx context.Context, transform func(string) string) (bool, error) {
		if running() {
			restarted, err := applyLibraryTransform(ctx, coordinator, config, transform)
			if !errors.Is(err, ErrUnsupportedAdapter) {
				return restarted, err
			}
		}
		return false, config.TransformRawSettings(transform)
	}
}

func applyLibraryTransform(ctx context.Context, coordinator *ApplyCoordinator, config *ConfigService, transform func(string) string) (bool, error) {
	caps := config.ConfigurationCapabilities()
	if !caps.Editable {
		return false, nil
	}
	live, err := config.GetRawSettings()
	if err != nil {
		return false, err
	}
	updated := transform(live.Content)
	if updated == live.Content {
		return false, nil
	}
	// npm already changed node_modules: a client disconnect must not roll the
	// settings back halfway through the restart.
	_, err = coordinator.Apply(context.WithoutCancel(ctx), ApplyRequest{
		Path: live.Path, Content: updated, Expected: live.Revision,
		BackupDir:    filepath.Join(config.dataDir, "backups", "settings"),
		Capabilities: caps,
	})
	return err == nil, err
}
