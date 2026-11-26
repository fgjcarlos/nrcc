package service

import (
	"context"
	"fmt"
)

// RecoverExternalApplyRuntime restarts the adapter against settings restored at boot.
func RecoverExternalApplyRuntime(ctx context.Context, req RollbackRequest) error {
	timeout := req.ReadyzTimeout
	if timeout <= 0 {
		timeout = max(defaultApplyRecoveryTimeout, readinessDeadline())
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if req.Stage == "" {
		req.Stage = RollbackStageRestart
	}
	emitRollback(req, "apply.rollback.started", "ok", nil, nil)
	if req.Adapter == nil {
		err := fmt.Errorf("apply recovery: runtime adapter is unresolved")
		emitRollback(req, "apply.rollback.failed", "error", err, map[string]string{"apply_rollback_action": "resolve"})
		return err
	}
	if err := req.Adapter.Restart(ctx); err != nil {
		emitRollback(req, "apply.rollback.failed", "error", err, map[string]string{"apply_rollback_action": "restart"})
		return fmt.Errorf("restart restored runtime: %w", err)
	}
	if err := req.Adapter.Ready(ctx); err != nil {
		emitRollback(req, "apply.rollback.failed", "error", err, map[string]string{"apply_rollback_action": "ready"})
		return fmt.Errorf("restored runtime readiness: %w", err)
	}
	emitRollback(req, "apply.rollback.succeeded", "ok", nil, nil)
	return nil
}
