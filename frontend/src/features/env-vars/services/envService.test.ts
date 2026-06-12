import { describe, expect, it, vi, beforeEach } from 'vitest';
import { envService, type EnvVar } from './envService';
import { api } from '@/shared/lib';

vi.mock('@/shared/lib', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

const mockGet = vi.mocked(api.get);
const mockPut = vi.mocked(api.put);

// Helper: the real backend wraps every payload in { success, data, timestamp }
// (see internal/model/response.go RespondJSON). axios exposes that body as
// `response.data`, so the inner payload lives at `response.data.data`.
function envelope<T>(data: T) {
  return { data: { success: true, data, timestamp: '2026-06-11T00:00:00Z' } };
}

describe('envService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getAll unwraps the envelope and returns the EnvVar list', async () => {
    const vars: EnvVar[] = [
      { key: 'FOO', value: '1', type: 'string' },
      { key: 'SECRET', value: 'x', type: 'secret', encrypted: true },
    ];
    mockGet.mockResolvedValueOnce(envelope(vars));

    const result = await envService.getAll();

    expect(mockGet).toHaveBeenCalledWith('/env');
    expect(result).toEqual(vars);
  });

  it('getDotenv unwraps the envelope and returns the .env content', async () => {
    mockGet.mockResolvedValueOnce(envelope({ content: 'FOO=1\nBAR=2' }));

    const result = await envService.getDotenv();

    expect(mockGet).toHaveBeenCalledWith('/env/dotenv');
    // The consumer reads result.content directly — it must be the inner payload,
    // not the full { success, data, timestamp } envelope.
    expect(result.content).toBe('FOO=1\nBAR=2');
  });

  it('saveDotenv unwraps the envelope and returns message + restarted', async () => {
    mockPut.mockResolvedValueOnce(envelope({ message: 'File saved', restarted: true }));

    const result = await envService.saveDotenv('FOO=1');

    expect(mockPut).toHaveBeenCalledWith('/env/dotenv', { content: 'FOO=1' });
    expect(result.message).toBe('File saved');
    expect(result.restarted).toBe(true);
  });
});
