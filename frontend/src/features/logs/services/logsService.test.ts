import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from '@/shared/lib/api';
import { logsService } from './logsService';

vi.mock('@/features/auth/services/authService', () => ({ authService: { getToken: () => 'test-token' } }));
vi.mock('@/shared/lib/api', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/shared/lib/api')>(),
  api: { get: vi.fn() },
}));

describe('logsService history and stream', () => {
  beforeEach(() => vi.clearAllMocks());
  afterEach(() => vi.unstubAllGlobals());

  it('walks every matching retained-history page under one generation and preserves snapshot metadata', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({ data: { data: {
      entries: [{ id: 'g:9', timestamp: '2025-01-01T00:00:09Z', level: 'error', source: 'r', message: 'match' }],
      generation: 'g', highWatermark: 'g:10', nextCursor: 'g:9', startedAt: '2025-01-01T00:00:00Z',
    } } }).mockResolvedValueOnce({ data: { data: {
      entries: [{ id: 'g:3', timestamp: '2025-01-01T00:00:03Z', level: 'error', source: 'r', message: 'match' }],
      generation: 'g', highWatermark: 'g:11',
    } } });
    const controller = new AbortController();
    const snapshots: string[] = [];
    const result = await logsService.history({ levels: ['error'], q: 'match' }, controller.signal, (page) => snapshots.push(page.highWatermark));

    expect(api.get).toHaveBeenCalledTimes(2);
    expect(String(vi.mocked(api.get).mock.calls[1][0])).toContain('cursor=g%3A9');
    expect(String(vi.mocked(api.get).mock.calls[1][0])).toContain('generation=g');
    expect(result.entries.map((entry) => entry.id)).toEqual(['g:3', 'g:9']);
    expect(result.highWatermark).toBe('g:10');
    expect(result.startedAt).toBe('2025-01-01T00:00:00.000Z');
    expect(snapshots).toEqual(['g:10']);
  });

  it('detects a runtime launch reported by SSE connected metadata and discards the old stream batch', async () => {
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new TextEncoder().encode(
          'event: connected\ndata: {"startedAt":"2025-01-01T00:05:00Z"}\n\n' +
          'id: g:11\nevent: log\ndata: {"id":"g:11","timestamp":"2025-01-01T00:05:01Z","level":"error","source":"runtime","message":"late old batch"}\n\n',
        ));
        controller.close();
      },
    });
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(body)));
    const controller = new AbortController();
    const onEntry = vi.fn();
    const onRestart = vi.fn(() => controller.abort());
    await logsService.stream('g:10', controller.signal, onEntry, onRestart, '2025-01-01T00:00:00Z');

    expect(onRestart).toHaveBeenCalledOnce();
    expect(onEntry).not.toHaveBeenCalled();
  });

  it('treats the zero time sentinel as an absent runtime start marker', async () => {
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new TextEncoder().encode('event: connected\ndata: {"startedAt":"0001-01-01T00:00:00Z"}\n\n'));
        controller.close();
      },
    });
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(body)));
    const onRestart = vi.fn();
    await logsService.stream('g:10', new AbortController().signal, vi.fn(), onRestart);
    expect(onRestart).not.toHaveBeenCalled();
  });
});
