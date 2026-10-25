import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as apiLib from '@/shared/lib';
import { configService, settingsService } from './index';

vi.mock('@/shared/lib', () => {
  const api = { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() };
  return { api, default: api };
});

const mockApi = (
  apiLib as unknown as { api: { get: ReturnType<typeof vi.fn>; post: ReturnType<typeof vi.fn> } }
).api;

beforeEach(() => vi.clearAllMocks());

describe('configService.updateConfig', () => {
  // Issue #878 — the visual save must carry the settings.js revision the
  // operator read so the backend can reject a stale edit.
  it('sends the expected revision in the request body', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await configService.updateConfig({ uiPort: 7777 }, 'rev-abc');

    expect(mockApi.post).toHaveBeenCalledWith('/config', {
      uiPort: 7777,
      expectedRevision: 'rev-abc',
    });
  });

  it('keeps the historical body shape when no revision is known', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await configService.updateConfig({ uiPort: 7777 });

    expect(mockApi.post).toHaveBeenCalledWith('/config', { uiPort: 7777 });
  });
});

describe('settingsService.saveRaw', () => {
  // Issue #878 — the raw editor sends the revision of the document it
  // opened, so a stale editor session cannot overwrite newer source.
  it('sends the expected revision in the request body', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await settingsService.saveRaw('module.exports = {};\n', 'rev-abc');

    expect(mockApi.post).toHaveBeenCalledWith('/settings/raw', {
      content: 'module.exports = {};\n',
      expectedRevision: 'rev-abc',
    });
  });

  it('keeps the historical body shape when no revision is known', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await settingsService.saveRaw('module.exports = {};\n');

    expect(mockApi.post).toHaveBeenCalledWith('/settings/raw', {
      content: 'module.exports = {};\n',
    });
  });
});
