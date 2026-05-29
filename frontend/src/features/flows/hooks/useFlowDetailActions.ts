import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { flowService } from '@/features/flows';
import type { AIFlowAction, FlowDetail } from '@/features/flows/types';
import { patternService, type PatternAnalysisResult } from '@/features/patterns/services';

export function useFlowDetailActions() {
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
    onError: (error: Error) => {
      toast.error(
        (error as Error).message || 'Failed to detect patterns'
      );
    },
  });

  const aiFlowMutation = useMutation({
    mutationFn: ({ action, flow, prompt }: { action: AIFlowAction; flow: FlowDetail; prompt?: string }) =>
      flowService.requestAIAssistance({
        action,
        prompt,
        flow: { id: flow.id, label: flow.label, nodes: flow.nodes },
      }),
    onSuccess: (data) => {
      toast.success(`AI ${data.action} response ready for review`);
    },
    onError: (error: Error) => {
      toast.error((error as Error).message || 'AI flow assistance is not available');
    },
  });

  return {
    analyzeFlowMutation,
    detectPatternsMutation,
    aiFlowMutation,
  };
}
