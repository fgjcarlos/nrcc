import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { flowService } from '@/features/flows';
import { AICapabilityUnavailableError } from '@/features/flows/services/flowService';
import type { AIFlowAction, FlowDetail } from '@/features/flows/types';

export function useFlowDetailActions() {
  // Analyze flow mutation
  const analyzeFlowMutation = useMutation({
    mutationFn: (flowId: string) => flowService.analyzeFlow(flowId),
    onSuccess: () => {
      toast.success('Flow analyzed successfully');
    },
    onError: (error: Error) => {
      toast.error(error instanceof AICapabilityUnavailableError ? error.message : 'Failed to analyze flow');
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
      toast.error(error.message || 'AI actions are unavailable until the configured provider is ready.');
    },
  });

  return {
    analyzeFlowMutation,
    aiFlowMutation,
  };
}
