import { describe, expect, it } from 'vitest';
import type { MetricsSnapshot, RestartEvent } from './history';

describe('MetricsSnapshot', () => {
  it('has the required shape with all numeric metric fields', () => {
    const snapshot: MetricsSnapshot = {
      timestamp: '2024-01-01T00:00:00Z',
      cpuPercent: 45.2,
      memoryPercent: 62.1,
      diskPercent: 78.5,
    };

    expect(snapshot.cpuPercent).toBe(45.2);
    expect(snapshot.memoryPercent).toBe(62.1);
    expect(snapshot.diskPercent).toBe(78.5);
    expect(snapshot.timestamp).toBe('2024-01-01T00:00:00Z');
  });
});

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
