/**
 * Change circuit (T9): maps what the product actually knows about a
 * configuration change onto the six contacts Edit → Diff → Validate →
 * Backup → Apply → Ready.
 *
 * The frontend only observes the start and the end of the server
 * transaction: POST /api/config answers once, after validate, backup,
 * write, restart and the readiness probe (internal/service/apply.go,
 * internal/service/apply_runtime.go). A failure carries the error code
 * that names its stage (internal/handler/config.go writeApplyError), so
 * every contact state below comes from one of those signals and never
 * from a guess. An answer without a known code leaves the server
 * contacts "unknown".
 */

/**
 * idle = not reached yet ("Waiting"); blocked = the gate holds and the
 * operator must act ("Open"); active = energising ("Pending").
 */
export type ContactState = 'idle' | 'blocked' | 'active' | 'passed' | 'failed' | 'unknown';

export type StepId = 'edit' | 'diff' | 'validate' | 'backup' | 'apply' | 'ready';

export const STEP_IDS: readonly StepId[] = ['edit', 'diff', 'validate', 'backup', 'apply', 'ready'];

/** i18n key under `configuration:circuit.proof` plus an optional count. */
export interface StepProof {
  key: string;
  count?: number;
}

export interface CircuitStep {
  id: StepId;
  state: ContactState;
  proof: StepProof | null;
  /** Server error code shown as the failed contact's designator. */
  code?: string;
}

export type ApplyRun =
  | { status: 'idle' }
  | { status: 'running'; count: number }
  | { status: 'succeeded'; count: number }
  | { status: 'failed'; count: number; code?: string };

export interface CircuitInput {
  pendingCount: number;
  invalidCount: number;
  /** The operator opened the review drawer for the current form state. */
  reviewed: boolean;
  readOnly: boolean;
  run: ApplyRun;
}

export interface Circuit {
  steps: CircuitStep[];
  /** Rollback branch from Apply back to the Backup contact, when taken. */
  rollback: 'restored' | 'failed' | null;
}

type FailedStage = 'validate' | 'backup' | 'write' | 'restart' | 'ready' | 'recovery' | 'unknown';

// Error codes written by internal/handler/config.go (SaveConfig,
// saveConfigViaApply, writeApplyError). Every one of the "validate"
// codes is returned before the backup, so nothing was written.
const STAGE_BY_CODE: Record<string, FailedStage> = {
  CONFIGURATION_READ_ONLY: 'validate',
  SETTINGS_ERROR: 'validate',
  SETTINGS_REVISION_CONFLICT: 'validate',
  APPLY_IN_FLIGHT: 'validate',
  SETTINGS_TIMEOUT: 'validate',
  VALIDATION_ERROR: 'validate',
  SETTINGS_BACKUP_FAILED: 'backup',
  SETTINGS_WRITE_ERROR: 'write',
  SETTINGS_RECONCILE_ERROR: 'write',
  // Restart and readiness failures are reported with these codes only
  // when ExecuteRollback restored the backup and Node-RED came back
  // ready (apply_runtime.go); a failed rollback is RUNTIME_RECOVERY_FAILED.
  RUNTIME_RESTART_FAILED: 'restart',
  RUNTIME_READINESS_FAILED: 'ready',
  RUNTIME_RECOVERY_FAILED: 'recovery',
};

/** Reads `{ response: { data: { error: { code } } } }` from an API error. */
export function applyErrorCode(error: unknown): string | undefined {
  const code = (error as { response?: { data?: { error?: { code?: unknown } } } } | null)
    ?.response?.data?.error?.code;
  return typeof code === 'string' ? code : undefined;
}

const step = (id: StepId, state: ContactState, proof: StepProof | null = null): CircuitStep => ({
  id,
  state,
  proof,
});

function serverSteps(run: ApplyRun, input: CircuitInput): CircuitStep[] {
  if (run.status === 'idle') {
    const anyChange = input.pendingCount > 0;
    const validate = !anyChange
      ? step('validate', 'idle')
      : input.invalidCount > 0
        ? step('validate', 'blocked', { key: 'invalid', count: input.invalidCount })
        : step('validate', 'idle', { key: 'localValid' });
    return [validate, step('backup', 'idle'), step('apply', 'idle'), step('ready', 'idle')];
  }
  if (run.status === 'running') {
    // One request covers all four stages; the order inside it is not observable.
    return [
      step('validate', 'active', { key: 'running' }),
      step('backup', 'active'),
      step('apply', 'active'),
      step('ready', 'active'),
    ];
  }
  if (run.status === 'succeeded') {
    return [
      step('validate', 'passed', { key: 'serverAccepted' }),
      step('backup', 'passed', { key: 'backupTaken' }),
      step('apply', 'passed', { key: 'written' }),
      step('ready', 'passed', { key: 'ready' }),
    ];
  }

  const stage: FailedStage = (run.code && STAGE_BY_CODE[run.code]) || 'unknown';
  const notReached = { key: 'notReached' };
  const failedAt = (id: StepId, key: string): CircuitStep => ({ ...step(id, 'failed', { key }), code: run.code });
  switch (stage) {
    case 'validate':
      return [failedAt('validate', 'rejected'), step('backup', 'idle', notReached), step('apply', 'idle', notReached), step('ready', 'idle', notReached)];
    case 'backup':
      return [step('validate', 'passed', { key: 'serverAccepted' }), failedAt('backup', 'backupFailed'), step('apply', 'idle', notReached), step('ready', 'idle', notReached)];
    case 'write':
      return [step('validate', 'passed', { key: 'serverAccepted' }), step('backup', 'passed', { key: 'backupTaken' }), failedAt('apply', 'writeFailed'), step('ready', 'idle', notReached)];
    case 'restart':
      return [step('validate', 'passed', { key: 'serverAccepted' }), step('backup', 'passed', { key: 'backupTaken' }), failedAt('apply', 'restartFailed'), step('ready', 'idle', notReached)];
    case 'ready':
      return [step('validate', 'passed', { key: 'serverAccepted' }), step('backup', 'passed', { key: 'backupTaken' }), step('apply', 'passed', { key: 'written' }), failedAt('ready', 'notReady')];
    case 'recovery':
      return [step('validate', 'passed', { key: 'serverAccepted' }), step('backup', 'passed', { key: 'backupTaken' }), failedAt('apply', 'recoveryFailed'), step('ready', 'unknown', { key: 'unknown' })];
    default:
      return [
        { ...step('validate', 'unknown', { key: 'unknown' }), code: run.code },
        step('backup', 'unknown'),
        step('apply', 'unknown'),
        step('ready', 'unknown'),
      ];
  }
}

export function deriveCircuit(input: CircuitInput): Circuit {
  const { run } = input;
  const inRun = run.status !== 'idle';
  const count = inRun ? run.count : input.pendingCount;

  const edit = inRun || count > 0
    ? step('edit', 'passed', { key: 'changes', count })
    : input.readOnly
      ? step('edit', 'blocked', { key: 'readOnly' })
      : step('edit', 'idle', { key: 'noChanges' });
  const diff = inRun || (count > 0 && input.reviewed)
    ? step('diff', 'passed', { key: 'reviewed' })
    : count > 0
      ? step('diff', 'active', { key: 'notReviewed' })
      : step('diff', 'idle');

  const stage = run.status === 'failed' ? (run.code && STAGE_BY_CODE[run.code]) || 'unknown' : null;
  const rollback = stage === 'restart' || stage === 'ready' ? 'restored' : stage === 'recovery' ? 'failed' : null;

  return { steps: [edit, diff, ...serverSteps(run, input)], rollback };
}
