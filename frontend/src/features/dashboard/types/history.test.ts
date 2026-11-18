import { describe, expect, it } from 'vitest';
import type { RestartEvent } from './history';

describe('RestartEvent', () => {
  it('has the required shape with exit code and attempt tracking', () => {
    const event: RestartEvent = {
      timestamp: '2024-01-01T00:00:00Z',
      exitCode: 1,
      attempt: 2,
      maxAttempts: 5,
    };

    expect(event.exitCode).toBe(1);
    expect(event.attempt).toBe(2);
    expect(event.maxAttempts).toBe(5);
    expect(event.timestamp).toBe('2024-01-01T00:00:00Z');
  });

  it('distinguishes clean exit from error exit via exitCode', () => {
    const cleanExit: RestartEvent = {
      timestamp: '2024-01-01T00:00:00Z',
      exitCode: 0,
      attempt: 1,
      maxAttempts: 3,
    };
    const errorExit: RestartEvent = {
      timestamp: '2024-01-01T00:01:00Z',
      exitCode: 137,
      attempt: 2,
      maxAttempts: 3,
    };

    expect(cleanExit.exitCode).toBe(0);
    expect(errorExit.exitCode).toBe(137);
    expect(cleanExit.exitCode).not.toBe(errorExit.exitCode);
  });
});
