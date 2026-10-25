package service

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const defaultApplyRecoveryTimeout = 30 * time.Second

// ApplyAdapterResolver selects the runtime authority once for a transaction.
type ApplyAdapterResolver func(ApplyRequest) (ApplyAdapter, error)

// NewRuntimeApplyCoordinator creates a coordinator that requires a supported runtime adapter.
func NewRuntimeApplyCoordinator(apply *ApplyService, resolve ApplyAdapterResolver) *ApplyCoordinator {
	coordinator := NewApplyCoordinator(apply)
	if resolve == nil {
		coordinator.resolve = func(ApplyRequest) (ApplyAdapter, error) {
			return nil, ErrUnsupportedAdapter
		}
		return coordinator
	}
	coordinator.resolve = resolve
	return coordinator
}

func (c *ApplyCoordinator) applyRuntime(ctx context.Context, req ApplyRequest) (ApplyResult, error) {
	adapter, err := c.resolve(req)
	if err != nil {
		c.apply.emit(req, "apply.failure", ApplyStageValidate, "error", err)
		return ApplyResult{}, &ApplyError{Stage: ApplyStageValidate, Cause: err, Path: req.Path, Request: req}
	}
	if adapter == nil {
		err = ErrUnsupportedAdapter
		c.apply.emit(req, "apply.failure", ApplyStageValidate, "error", err)
		return ApplyResult{}, &ApplyError{Stage: ApplyStageValidate, Cause: err, Path: req.Path, Request: req}
	}
	req.runtimeAdapter = adapter
	state, err := c.stageStructured(&req)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := c.apply.applyValidate(ctx, req); err != nil {
		if state != nil {
			if restoreErr := c.apply.configSvc.RestoreStructured(*state); restoreErr != nil {
				err = errors.Join(err, fmt.Errorf("restore structured config: %w", restoreErr))
			}
		}
		return ApplyResult{}, &ApplyError{Stage: ApplyStageValidate, Cause: err, Path: req.Path, Request: req}
	}
	result, err := c.apply.apply(ctx, req, false)
	if err != nil {
		if state != nil {
			if restoreErr := c.apply.configSvc.RestoreStructured(*state); restoreErr != nil {
				return ApplyResult{}, &ApplyError{Stage: ApplyStageRollback, Cause: errors.Join(err, fmt.Errorf("restore structured config: %w", restoreErr)), Path: req.Path, Request: req}
			}
		}
		return ApplyResult{}, err
	}
	if state != nil {
		result.Configuration = state.Committed
	}
	stage := ApplyStageRestart
	cause := adapter.Restart(ctx)
	if cause == nil {
		if ctx.Err() != nil {
			cause = ctx.Err()
		} else {
			stage = ApplyStageReady
			cause = adapter.Ready(ctx)
			if cause == nil && ctx.Err() != nil {
				cause = ctx.Err()
			}
		}
	}
	if cause != nil {
		original := &ApplyError{Stage: stage, Cause: cause, Path: req.Path, Request: req}
		recoveryTimeout := req.RecoveryTimeout
		if recoveryTimeout <= 0 {
			recoveryTimeout = defaultApplyRecoveryTimeout
		}
		recoveryCtx, cancel := context.WithTimeout(context.Background(), recoveryTimeout)
		recoveryErr := ExecuteRollback(recoveryCtx, RollbackRequest{
			BackupPath: result.BackupPath, ActivePath: req.Path, Stage: RollbackStage(stage),
			RestoreState: func(context.Context) error {
				if state == nil {
					return nil
				}
				return c.apply.configSvc.RestoreStructured(*state)
			},
			Adapter: adapter, Actor: req.Actor, Request: req.Request, AuditHook: c.apply.auditHook,
			Capabilities: req.Capabilities, ReadyzTimeout: recoveryTimeout,
		})
		cancel()
		if recoveryErr != nil {
			original.Cause = errors.Join(cause, fmt.Errorf("recovery failed: %w", recoveryErr))
		}
		c.apply.emit(req, "apply.failure", stage, "error", original.Cause)
		return ApplyResult{}, original
	}
	c.apply.emit(req, "apply.success", ApplyStageReady, "ok", nil)
	return result, nil
}
