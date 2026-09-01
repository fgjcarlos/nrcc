import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as apiLib from '@/shared/lib';
import { aiService } from './aiService';

vi.mock('@/shared/lib', () => ({ api: { get: vi.fn(), put: vi.fn(), post: vi.fn() } }));

const mockApi = (apiLib as unknown as { api: { get: ReturnType<typeof vi.fn>; put: ReturnType<typeof vi.fn>; post: ReturnType<typeof vi.fn> } }).api;
const ok = <T>(data: T) => ({ data: { data } });

beforeEach(() => vi.clearAllMocks());

describe('aiService', () => {
  it('reads the safe configuration and capability contracts', async () => {
    const config = { enabled: true, provider: 'openai' as const, apiKeyConfigured: true };
    const status = { status: 'ready' as const, provider: 'openai' as const };
    mockApi.get.mockResolvedValueOnce(ok(config)).mockResolvedValueOnce(ok(status));

    await expect(aiService.getConfig()).resolves.toEqual(config);
    await expect(aiService.getStatus()).resolves.toEqual(status);
    expect(mockApi.get).toHaveBeenNthCalledWith(1, '/ai/config');
    expect(mockApi.get).toHaveBeenNthCalledWith(2, '/ai/status');
  });

  it('saves write-only API keys and starts a connection test', async () => {
    const config = { enabled: true, provider: 'openai' as const, endpoint: 'https://api.example.test', model: 'model', apiKey: 'secret' };
    mockApi.put.mockResolvedValueOnce(ok({ ...config, apiKeyConfigured: true }));
    mockApi.post.mockResolvedValueOnce(ok({ status: 'ready', provider: 'openai' }));

    await aiService.saveConfig(config);
    await aiService.testConfig();
    expect(mockApi.put).toHaveBeenCalledWith('/ai/config', config);
    expect(mockApi.post).toHaveBeenCalledWith('/ai/config/test');
  });
});
