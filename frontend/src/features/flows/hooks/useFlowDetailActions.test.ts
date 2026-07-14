import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { useFlowDetailActions } from './useFlowDetailActions';

// ─── Mocks ────────────────────────────────────────────────────────────────────

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}));

vi.mock('@/features/flows', () => ({
  flowService: {
    analyzeFlow: vi.fn(),
    requestAIAssistance: vi.fn(),
  },
}));

import { toast } from 'sonner';
import { flowService } from '@/features/flows';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
}

// ─── Tests ────────────────────────────────────────────────────────────────────

beforeEach(() => vi.clearAllMocks());

describe('useFlowDetailActions', () => {
  describe('analyzeFlowMutation', () => {
    it('calls flowService.analyzeFlow with the provided flowId', async () => {
      const analysis = { flowId: 'f1', summary: 'ok', pros: [], cons: [], suggestions: [], analyzedAt: '' };
      vi.mocked(flowService.analyzeFlow).mockResolvedValueOnce(analysis);

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.analyzeFlowMutation.mutate('f1');
        await vi.waitFor(() => !result.current.analyzeFlowMutation.isPending);
      });

      expect(flowService.analyzeFlow).toHaveBeenCalledWith('f1');
      expect(toast.success).toHaveBeenCalledWith('Flow analyzed successfully');
    });

    it('shows a generic error toast when analyzeFlow fails', async () => {
      vi.mocked(flowService.analyzeFlow).mockRejectedValueOnce(new Error('server error'));

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.analyzeFlowMutation.mutate('f1');
        await vi.waitFor(() => result.current.analyzeFlowMutation.isError);
      });

      expect(toast.error).toHaveBeenCalledWith('Failed to analyze flow');
    });
  });

  describe('aiFlowMutation', () => {
    it('calls flowService.requestAIAssistance and shows a success toast', async () => {
      const aiResponse = {
        enabled: true,
        provider: 'openai',
        action: 'explain' as const,
        reviewOnly: true,
        redacted: true,
        summary: 'Here is the explanation',
      };
      vi.mocked(flowService.requestAIAssistance).mockResolvedValueOnce(aiResponse);

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });
      const flow = { id: 'f1', label: 'Flow 1', nodes: [] };

      await act(async () => {
        result.current.aiFlowMutation.mutate({ action: 'explain', flow });
        await vi.waitFor(() => !result.current.aiFlowMutation.isPending);
      });

      expect(flowService.requestAIAssistance).toHaveBeenCalledWith({
        action: 'explain',
        prompt: undefined,
        flow: { id: 'f1', label: 'Flow 1', nodes: [] },
      });
      expect(toast.success).toHaveBeenCalledWith('AI explain response ready for review');
    });

    it('shows an error toast when AI assistance fails', async () => {
      vi.mocked(flowService.requestAIAssistance).mockRejectedValueOnce(new Error('AI unavailable'));

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });
      const flow = { id: 'f1', label: 'Flow 1', nodes: [] };

      await act(async () => {
        result.current.aiFlowMutation.mutate({ action: 'audit', flow });
        await vi.waitFor(() => result.current.aiFlowMutation.isError);
      });

      expect(toast.error).toHaveBeenCalledWith('AI unavailable');
    });
  });
});
