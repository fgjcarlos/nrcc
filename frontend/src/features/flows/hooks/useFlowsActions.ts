import { useCallback } from 'react';
import { flowService, type AnalysisResult } from '@/features/flows';
import { AICapabilityUnavailableError } from '@/features/flows/services/flowService';
import { toast } from 'sonner';

export interface UseFlowsActionsResult {
  analyzeFlows: (ids: string[]) => Promise<Record<string, AnalysisResult>>;
}

export function useFlowsActions(): UseFlowsActionsResult {
  const analyzeFlows = useCallback(
    async (ids: string[]): Promise<Record<string, AnalysisResult>> => {
      if (ids.length === 0) {
        return {};
      }

      const results = await Promise.allSettled(
        ids.map(id => flowService.analyzeFlow(id))
      );

      const newResults: Record<string, AnalysisResult> = {};
      let failed = 0;

      results.forEach((result, i) => {
        if (result.status === 'fulfilled') {
          newResults[ids[i]] = result.value;
        } else {
          failed++;
        }
      });

      const unavailable = results.find(
        (result): result is PromiseRejectedResult =>
          result.status === 'rejected' && result.reason instanceof AICapabilityUnavailableError,
      );

      if (unavailable && failed === ids.length) {
        toast.error(unavailable.reason.message);
      } else if (failed > 0) {
        toast.error(
          `${failed} flow${failed > 1 ? 's' : ''} failed to analyze. Check AI configuration.`
        );
      } else {
        toast.success(
          `${ids.length} flow${ids.length > 1 ? 's' : ''} analyzed successfully`
        );
      }

      return newResults;
    },
    []
  );

  return {
    analyzeFlows,
  };
}
