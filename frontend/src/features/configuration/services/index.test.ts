import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as apiLib from '@/shared/lib';
import { CONFIG_APPLY_TIMEOUT_MS, configService, settingsService } from './index';

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

    expect(mockApi.post).toHaveBeenCalledWith(
      '/config',
      { uiPort: 7777, expectedRevision: 'rev-abc' },
      { timeout: CONFIG_APPLY_TIMEOUT_MS },
    );
  });

  it('keeps the historical body shape when no revision is known', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await configService.updateConfig({ uiPort: 7777 });

    expect(mockApi.post).toHaveBeenCalledWith(
      '/config',
      { uiPort: 7777 },
      { timeout: CONFIG_APPLY_TIMEOUT_MS },
    );
  });
});

describe('settingsService.saveRaw', () => {
  // Issue #878 — the raw editor sends the revision of the document it
  // opened, so a stale editor session cannot overwrite newer source.
  it('sends the expected revision in the request body', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await settingsService.saveRaw('module.exports = {};\n', 'rev-abc');

    expect(mockApi.post).toHaveBeenCalledWith(
      '/settings/raw',
      { content: 'module.exports = {};\n', expectedRevision: 'rev-abc' },
      { timeout: CONFIG_APPLY_TIMEOUT_MS },
    );
  });

  it('keeps the historical body shape when no revision is known', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await settingsService.saveRaw('module.exports = {};\n');

    expect(mockApi.post).toHaveBeenCalledWith(
      '/settings/raw',
      { content: 'module.exports = {};\n' },
      { timeout: CONFIG_APPLY_TIMEOUT_MS },
    );
  });
});

// The visual save, the transactional apply and the raw save all write
// settings.js and wait for Node-RED to restart and pass readiness before
// they answer. That outlasts the 10 s axios default; an aborted request
// cancels the transaction server-side and the save is rolled back.
describe('requests that restart Node-RED', () => {
  it('outlast the backend readiness budget', () => {
    expect(CONFIG_APPLY_TIMEOUT_MS).toBe(120_000);
  });

  it('configService.updateConfig sends the apply timeout', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await configService.updateConfig({ uiPort: 7777 }, 'rev-abc');

    expect(mockApi.post.mock.calls[0][2]).toEqual({ timeout: 120_000 });
  });

  it('configService.applyConfig sends the apply timeout', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await configService.applyConfig({ uiPort: 7777 }, 'rev-abc');

    expect(mockApi.post).toHaveBeenCalledWith(
      '/config/apply',
      { uiPort: 7777, expectedRevision: 'rev-abc' },
      { timeout: 120_000 },
    );
  });

  it('settingsService.saveRaw sends the apply timeout', async () => {
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await settingsService.saveRaw('module.exports = {};\n', 'rev-abc');

    expect(mockApi.post.mock.calls[0][2]).toEqual({ timeout: 120_000 });
  });

  it('leave read and validation requests on the default timeout', async () => {
    mockApi.get.mockResolvedValue({ data: { data: {} } });
    mockApi.post.mockResolvedValueOnce({ data: { data: {} } });

    await configService.getConfig();
    await settingsService.getRaw();
    await configService.validateConfig({});

    for (const call of [...mockApi.get.mock.calls, ...mockApi.post.mock.calls]) {
      expect(call[2]).toBeUndefined();
      expect(call.length).toBeLessThanOrEqual(2);
    }
  });
});
