import { describe, expect, it } from 'vitest';
import { deriveCircuit, isCircuitAtRest } from './changeCircuit';

const base = { pendingCount: 2, invalidFields: [], reviewed: true, readOnly: false };

describe('deriveCircuit — gated outcomes stay on the circuit', () => {
  it('holds Validate with the invalid keys when the save gate rejects fields', () => {
    const { steps, rollback } = deriveCircuit({
      ...base,
      run: { status: 'held', count: 2, reason: 'invalid', fields: ['uiPort', 'https.port'] },
    });
    expect(steps.map((s) => s.state)).toEqual(['passed', 'passed', 'blocked', 'idle', 'idle', 'idle']);
    expect(steps[2].proof).toEqual({ key: 'invalid', count: 2 });
    expect(steps[2].code).toBe('uiPort, https.port');
    expect(rollback).toBeNull();
  });

  it('blocks Edit when the save gate reports a read-only runtime', () => {
    const { steps } = deriveCircuit({ ...base, run: { status: 'held', count: 2, reason: 'readOnly' } });
    expect(steps[0]).toMatchObject({ state: 'blocked', proof: { key: 'readOnly' } });
    expect(steps.slice(2).every((s) => s.state === 'idle')).toBe(true);
  });
});

describe('isCircuitAtRest — the compact idle line', () => {
  const rest = { pendingCount: 0, invalidFields: [], reviewed: false, readOnly: false, run: { status: 'idle' } as const };

  it('is at rest with no change, no run and no outcome', () => {
    expect(isCircuitAtRest(deriveCircuit(rest))).toBe(true);
  });

  it('is not at rest once there is a change, a review, a run, an outcome or a held gate', () => {
    expect(isCircuitAtRest(deriveCircuit({ ...rest, pendingCount: 1 }))).toBe(false);
    expect(isCircuitAtRest(deriveCircuit({ ...rest, pendingCount: 1, reviewed: true }))).toBe(false);
    expect(isCircuitAtRest(deriveCircuit({ ...rest, run: { status: 'running', count: 1 } }))).toBe(false);
    expect(isCircuitAtRest(deriveCircuit({ ...rest, run: { status: 'succeeded', count: 1 } }))).toBe(false);
    expect(isCircuitAtRest(deriveCircuit({ ...rest, run: { status: 'failed', count: 1 } }))).toBe(false);
    expect(isCircuitAtRest(deriveCircuit({ ...rest, readOnly: true }))).toBe(false);
  });
});
