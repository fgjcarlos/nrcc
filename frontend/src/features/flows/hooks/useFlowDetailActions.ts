import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { flowService, type AnalysisResult } from '@/features/flows';
import { patternService, type PatternAnalysisResult } from '@/features/patterns/services';

export function useFlowDetailActions() {
  const queryClient = useQueryClient();

  // Analyze flow mutation
  const analyzeFlowMutation = useMutation({
    mutationFn: (flowId: string) => flowService.analyzeFlow(flowId),
    onSuccess: () => {
      toast.success('Flow analyzed successfully');
    },
    onError: () => {
      toast.error('Failed to analyze flow');
    },
  });

  // Detect patterns mutation
  const detectPatternsMutation = useMutation({
    mutationFn: (flowIds: string[]) =>
      patternService.analyzePatterns({ flowIds }),
    onSuccess: (data: PatternAnalysisResult) => {
      if (data.patterns.length === 0) {
        toast.info(
          data.message || 'No patterns detected in the selected flows'
        );
      } else {
        toast.success(
          `Found ${data.patterns.length} pattern${
            data.patterns.length !== 1 ? 's' : ''
          }`
        );
      }
    },
    onError: (error) => {
      toast.error(
        (error as Error).message || 'Failed to detect patterns'
      );
    },
  });

  return {
    analyzeFlowMutation,
    detectPatternsMutation,
  };
}
