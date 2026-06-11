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

vi.mock('@/features/patterns/services', () => ({
  patternService: {
    analyzePatterns: vi.fn(),
  },
}));

import { toast } from 'sonner';
import { flowService } from '@/features/flows';
import { patternService } from '@/features/patterns/services';

// ─── Helpers ──────────────────────────────────────────────────────────────────

function makeAxiosError(status: number): Error {
  const err = new Error(`Request failed with status code ${status}`) as Error & {
    isAxiosError: boolean;
    response: { status: number };
  };
  err.isAxiosError = true;
  err.response = { status };
  return err;
}

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

  describe('detectPatternsMutation', () => {
    it('calls patternService.analyzePatterns with the provided flowIds', async () => {
      vi.mocked(patternService.analyzePatterns).mockResolvedValueOnce({
        patternId: 'p1',
        patterns: [{ id: 'pat1', name: 'P', description: '', frequency: 2, flows: [], nodeSuggestion: { name: 'n', category: 'c', inputs: 1, outputs: 1, properties: [] } }],
        analyzedAt: '',
        flowCount: 2,
      });

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.detectPatternsMutation.mutate(['f1', 'f2']);
        await vi.waitFor(() => !result.current.detectPatternsMutation.isPending);
      });

      expect(patternService.analyzePatterns).toHaveBeenCalledWith({ flowIds: ['f1', 'f2'] });
      expect(toast.success).toHaveBeenCalledWith('Found 1 pattern');
    });

    it('shows a plural success toast when multiple patterns are found', async () => {
      vi.mocked(patternService.analyzePatterns).mockResolvedValueOnce({
        patternId: 'p1',
        patterns: [
          { id: 'a', name: 'A', description: '', frequency: 1, flows: [], nodeSuggestion: { name: 'n', category: 'c', inputs: 1, outputs: 1, properties: [] } },
          { id: 'b', name: 'B', description: '', frequency: 1, flows: [], nodeSuggestion: { name: 'n', category: 'c', inputs: 1, outputs: 1, properties: [] } },
        ],
        analyzedAt: '',
        flowCount: 2,
      });

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.detectPatternsMutation.mutate(['f1', 'f2']);
        await vi.waitFor(() => !result.current.detectPatternsMutation.isPending);
      });

      expect(toast.success).toHaveBeenCalledWith('Found 2 patterns');
    });

    it('shows an info toast when no patterns are found', async () => {
      vi.mocked(patternService.analyzePatterns).mockResolvedValueOnce({
        patternId: 'p1',
        patterns: [],
        analyzedAt: '',
        flowCount: 2,
        message: 'Nothing to see here',
      });

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.detectPatternsMutation.mutate(['f1', 'f2']);
        await vi.waitFor(() => !result.current.detectPatternsMutation.isPending);
      });

      expect(toast.info).toHaveBeenCalledWith('Nothing to see here');
    });

    it('shows "coming soon" info toast when the endpoint returns 501 NOT_IMPLEMENTED', async () => {
      vi.mocked(patternService.analyzePatterns).mockRejectedValueOnce(makeAxiosError(501));

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.detectPatternsMutation.mutate(['f1']);
        await vi.waitFor(() => result.current.detectPatternsMutation.isError);
      });

      expect(toast.info).toHaveBeenCalledWith(
        'Pattern detection is not yet available — coming soon'
      );
      expect(toast.error).not.toHaveBeenCalled();
    });

    it('shows a generic error toast for non-501 errors', async () => {
      vi.mocked(patternService.analyzePatterns).mockRejectedValueOnce(makeAxiosError(500));

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.detectPatternsMutation.mutate(['f1']);
        await vi.waitFor(() => result.current.detectPatternsMutation.isError);
      });

      expect(toast.error).toHaveBeenCalled();
      expect(toast.info).not.toHaveBeenCalledWith(
        'Pattern detection is not yet available — coming soon'
      );
    });

    it('shows a generic error toast for plain network errors (no status code)', async () => {
      vi.mocked(patternService.analyzePatterns).mockRejectedValueOnce(new Error('connection refused'));

      const { result } = renderHook(() => useFlowDetailActions(), { wrapper: createWrapper() });

      await act(async () => {
        result.current.detectPatternsMutation.mutate(['f1']);
        await vi.waitFor(() => result.current.detectPatternsMutation.isError);
      });

      expect(toast.error).toHaveBeenCalledWith('connection refused');
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
