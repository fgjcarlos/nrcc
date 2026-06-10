import { useQuery, type QueryObserverResult } from '@tanstack/react-query';
import { flowService } from '@/features/flows';
import type { FlowSummary } from '@/features/flows/types';

import { queryKeys } from '@/shared/lib/queryKeys';

type FlowsResponse = { available: boolean; flows: FlowSummary[] };

export interface UseFlowsDataResult {
  flows: FlowSummary[];
  available: boolean;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<QueryObserverResult<FlowsResponse, Error>>;
}

export function useFlowsData(): UseFlowsDataResult {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: queryKeys.flows.root,
    queryFn: flowService.getFlows,
    refetchInterval: 30000,
  });

  return {
    flows: data?.flows ?? [],
    available: data?.available ?? false,
    isLoading,
    error: error as Error | null,
    refetch,
  };
}
