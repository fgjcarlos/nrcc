import { useQuery } from '@tanstack/react-query';
import { flowService } from '@/features/flows';

import { queryKeys } from '@/shared/lib/queryKeys';
interface UseFlowDetailDataParams {
  flowId: string | undefined;
}

export function useFlowDetailData({ flowId }: UseFlowDetailDataParams) {
  // Flow detail query
  const flowQuery = useQuery({
    queryKey: queryKeys.flows.detail(flowId),
    queryFn: () => flowService.getFlowById(flowId!),
    enabled: !!flowId,
  });

  // Flow metrics query
  const metricsQuery = useQuery({
    queryKey: queryKeys.flows.metrics(flowId),
    queryFn: () => flowService.getFlowMetrics(flowId!),
    enabled: !!flowId,
  });

  // All flows query (for pattern detection selector)
  const allFlowsQuery = useQuery({
    queryKey: queryKeys.flows.root,
    queryFn: flowService.getFlows,
  });

  return {
    flow: flowQuery.data,
    flowLoading: flowQuery.isLoading,
    flowError: flowQuery.isError,

    metrics: metricsQuery.data,
    metricsLoading: metricsQuery.isLoading,
    metricsError: metricsQuery.isError,

    allFlows: allFlowsQuery.data,
    allFlowsLoading: allFlowsQuery.isLoading,
    allFlowsError: allFlowsQuery.isError,

    // Combined loading state
    isLoading: flowQuery.isLoading || metricsQuery.isLoading,

    // Refetch functions
    refetchFlow: flowQuery.refetch,
    refetchMetrics: metricsQuery.refetch,
    refetchAllFlows: allFlowsQuery.refetch,
  };
}
