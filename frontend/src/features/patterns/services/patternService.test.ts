import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from 'shared/lib/api';
import { authService } from '@/features/auth/services/authService';
import { patternService } from './patternService';

// The pattern endpoints are documented in docs/openapi.yaml and mounted by the
// Go server under /api/ai/* (see internal/server/server.go). The axios instance
// already prefixes /api, so the service must call the /ai/* paths; downloadReadme
// uses raw fetch and therefore needs the full /api/ai/* path. These tests pin the
// URLs to the spec so the UI works the day the 501 stub becomes a real handler.
vi.mock('shared/lib/api', () => ({
  api: {
    post: vi.fn(),
    get: vi.fn(),
  },
}));

vi.mock('@/features/auth/services/authService', () => ({
  authService: {
    getToken: vi.fn(() => 'test-token'),
  },
}));

const mockPost = vi.mocked(api.post);
const mockGet = vi.mocked(api.get);

beforeEach(() => {
  vi.clearAllMocks();
});

describe('patternService URL contract (OpenAPI alignment)', () => {
  it('analyzePatterns POSTs to /ai/analyze/patterns', async () => {
    mockPost.mockResolvedValueOnce({
      data: { data: { patternId: 'p1', patterns: [], analyzedAt: '', flowCount: 0 } },
    } as never);

    await patternService.analyzePatterns({ flowIds: ['a', 'b'] });

    expect(mockPost).toHaveBeenCalledWith(
      '/ai/analyze/patterns',
      { flowIds: ['a', 'b'] },
      { timeout: 60000 }
    );
  });

  it('getReadme GETs /ai/patterns/{analysisId}/readme with the patternId query', async () => {
    mockGet.mockResolvedValueOnce({
      data: { data: { readme: '# hi', patternName: 'P' } },
    } as never);

    await patternService.getReadme('analysis-1', 'pattern-9');

    expect(mockGet).toHaveBeenCalledWith(
      '/ai/patterns/analysis-1/readme?patternId=pattern-9'
    );
  });
});

describe('patternService.downloadReadme', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
    // jsdom lacks object-URL plumbing; stub so the happy path does not throw.
    vi.stubGlobal('URL', {
      ...URL,
      createObjectURL: vi.fn(() => 'blob:fake'),
      revokeObjectURL: vi.fn(),
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('fetches the full /api/ai/patterns/{analysisId}/download path with the bearer token', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      blob: async () => new Blob(['readme']),
      headers: { get: () => 'attachment; filename="pattern.md"' },
    } as unknown as Response);

    await patternService.downloadReadme('analysis-1', 'pattern-9');

    expect(authService.getToken).toHaveBeenCalled();
    expect(fetch).toHaveBeenCalledWith(
      '/api/ai/patterns/analysis-1/download?patternId=pattern-9',
      expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({ Authorization: 'Bearer test-token' }),
      })
    );
  });
});
