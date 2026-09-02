import { describe, it, expect, vi, beforeEach } from 'vitest';
import { flowService } from './flowService';
import * as apiLib from '@/shared/lib';

vi.mock('@/shared/lib', () => ({
  default: { get: vi.fn(), post: vi.fn() },
  api: { get: vi.fn(), post: vi.fn() },
}));

// flowService imports { api } from '@/shared/lib' (named export)
const mockApi = (apiLib as unknown as { api: { get: ReturnType<typeof vi.fn>; post: ReturnType<typeof vi.fn> } }).api;

const ok = <T>(data: T) => ({ data: { data } });
const readyCapability = { status: 'ready' as const, provider: 'openai' as const, model: 'model' };

beforeEach(() => vi.clearAllMocks());

describe('flowService', () => {
  describe('getAICapability', () => {
    it('calls the authoritative AI capability endpoint', async () => {
      const capability = { status: 'unreachable' as const, provider: 'openai' as const, model: 'model', reason: 'The provider could not be reached' };
      mockApi.get.mockResolvedValueOnce(ok(capability));

      const result = await flowService.getAICapability();

      expect(mockApi.get).toHaveBeenCalledWith('/ai/status');
      expect(result).toEqual(capability);
    });
  });

  describe('getFlows', () => {
    it('calls GET /flows and returns the data envelope', async () => {
      const payload = { available: true, flows: [{ id: 'f1', label: 'Flow 1', nodes: 3, connections: 2, disabled: false }] };
      mockApi.get.mockResolvedValueOnce(ok(payload));

      const result = await flowService.getFlows();

      expect(mockApi.get).toHaveBeenCalledWith('/flows');
      expect(result).toEqual(payload);
    });

    it('propagates API errors', async () => {
      mockApi.get.mockRejectedValueOnce(new Error('network error'));
      await expect(flowService.getFlows()).rejects.toThrow('network error');
    });
  });

  describe('getFlowById', () => {
    it('calls GET /flows/:id', async () => {
      const flow = { id: 'f1', label: 'Flow 1', nodes: [] };
      mockApi.get.mockResolvedValueOnce(ok(flow));

      const result = await flowService.getFlowById('f1');

      expect(mockApi.get).toHaveBeenCalledWith('/flows/f1');
      expect(result).toEqual(flow);
    });
  });

  describe('getFlowMetrics', () => {
    it('calls GET /flows/:id/metrics', async () => {
      const metrics = { nodeCount: 5, connectionCount: 3, entryPoints: [], exitPoints: [], nodeTypes: {}, disabledNodes: 0 };
      mockApi.get.mockResolvedValueOnce(ok(metrics));

      const result = await flowService.getFlowMetrics('f1');

      expect(mockApi.get).toHaveBeenCalledWith('/flows/f1/metrics');
      expect(result).toEqual(metrics);
    });
  });

  describe('analyzeFlow', () => {
    it('loads the real flow and sends it to the existing AI audit endpoint', async () => {
      const flow = { id: 'f1', label: 'Flow 1', nodes: [] };
      mockApi.get.mockResolvedValueOnce(ok(readyCapability));
      mockApi.get.mockResolvedValueOnce(ok(flow));
      mockApi.get.mockResolvedValueOnce(ok(readyCapability));
      mockApi.post.mockResolvedValueOnce(ok({
        enabled: true,
        provider: 'openai',
        action: 'audit',
        reviewOnly: true,
        redacted: true,
        summary: 'ok',
        suggestions: ['add retry'],
        auditFindings: ['missing catch'],
      }));

      const result = await flowService.analyzeFlow('f1');

      expect(mockApi.get).toHaveBeenNthCalledWith(1, '/ai/status');
      expect(mockApi.get).toHaveBeenNthCalledWith(2, '/flows/f1');
      expect(mockApi.get).toHaveBeenNthCalledWith(3, '/ai/status');
      expect(mockApi.post).toHaveBeenCalledWith('/ai/analyze/flow', {
        action: 'audit',
        flow,
      });
      expect(result).toMatchObject({
        flowId: 'f1',
        summary: 'ok',
        pros: [],
        cons: ['missing catch'],
        suggestions: ['add retry'],
      });
    });

    it('fails closed before loading a flow when the provider is not ready', async () => {
      mockApi.get.mockResolvedValueOnce(ok({
        status: 'unreachable' as const,
        provider: 'openai' as const,
        model: 'model',
        reason: 'The provider could not be reached',
      }));

      await expect(flowService.analyzeFlow('f1')).rejects.toThrow('The provider could not be reached');

      expect(mockApi.get).toHaveBeenCalledWith('/ai/status');
      expect(mockApi.get).not.toHaveBeenCalledWith('/flows/f1');
      expect(mockApi.post).not.toHaveBeenCalled();
    });
  });

  describe('requestAIAssistance', () => {
    it('calls POST /ai/analyze/flow with the input payload', async () => {
      const aiResponse = {
        enabled: true,
        provider: 'openai',
        action: 'explain' as const,
        reviewOnly: true,
        redacted: true,
        summary: 'Explanation here',
      };
      mockApi.get.mockResolvedValueOnce(ok(readyCapability));
      mockApi.post.mockResolvedValueOnce(ok(aiResponse));

      const input = {
        action: 'explain' as const,
        flow: { id: 'f1', label: 'Flow 1', nodes: [] },
        prompt: 'Explain this',
      };
      const result = await flowService.requestAIAssistance(input);

      expect(mockApi.post).toHaveBeenCalledWith('/ai/analyze/flow', input);
      expect(result).toEqual(aiResponse);
    });

    it('blocks a direct copilot request when a fresh capability check is not ready', async () => {
      mockApi.get.mockResolvedValueOnce(ok({
        status: 'disabled' as const,
        provider: 'offline' as const,
        model: '',
        reason: 'AI assistance is disabled',
      }));

      await expect(flowService.requestAIAssistance({
        action: 'explain',
        flow: { id: 'f1', label: 'Flow 1', nodes: [] },
      })).rejects.toThrow('AI assistance is disabled');

      expect(mockApi.post).not.toHaveBeenCalled();
    });
  });

  describe('getVersions', () => {
    it('calls GET /flows/versions', async () => {
      const versions = [{ id: 'v1', timestamp: '2026-01-01T00:00:00Z', hash: 'abc', nodeCount: 3, size: 512 }];
      mockApi.get.mockResolvedValueOnce(ok(versions));

      const result = await flowService.getVersions();

      expect(mockApi.get).toHaveBeenCalledWith('/flows/versions');
      expect(result).toEqual(versions);
    });
  });

  describe('getVersionDiff', () => {
    it('calls GET /flows/versions/:from/diff/:to', async () => {
      const diff = { added: [], removed: [], modified: [] };
      mockApi.get.mockResolvedValueOnce(ok(diff));

      const result = await flowService.getVersionDiff('v1', 'v2');

      expect(mockApi.get).toHaveBeenCalledWith('/flows/versions/v1/diff/v2');
      expect(result).toEqual(diff);
    });
  });

  describe('revertToVersion', () => {
    it('calls POST /flows/versions/:id/revert', async () => {
      mockApi.post.mockResolvedValueOnce({});

      await flowService.revertToVersion('v1');

      expect(mockApi.post).toHaveBeenCalledWith('/flows/versions/v1/revert');
    });
  });

  describe('captureSnapshot', () => {
    it('calls POST /flows/versions', async () => {
      mockApi.post.mockResolvedValueOnce({});

      await flowService.captureSnapshot();

      expect(mockApi.post).toHaveBeenCalledWith('/flows/versions');
    });
  });
});
