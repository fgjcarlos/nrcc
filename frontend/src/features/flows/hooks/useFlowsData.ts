import { useQuery } from '@tanstack/react-query';
import { flowService } from '@/features/flows';

export interface UseFlowsDataResult {
  flows: any[];
  available: boolean;
  isLoading: boolean;
  error: Error | null;
  refetch: () => Promise<any>;
}

export function useFlowsData(): UseFlowsDataResult {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['flows'],
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
