import { describe, expect, it } from 'vitest';
import { deriveCircuit } from './changeCircuit';

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
