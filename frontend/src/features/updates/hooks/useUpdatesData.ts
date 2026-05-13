import { useQuery } from '@tanstack/react-query';
import { updateService } from '@/features/updates/services';
import { UPDATE_STATUS_KEY } from './useUpdateStatus';
import { UPDATE_FLOW_STATE_KEY } from './useUpdateFlowState';

/**
 * Hook to fetch all update-related queries: status, flow state, and history.
 * Combines polling for real-time status and history tracking.
 */
export function useUpdatesData() {
  // Status query with 30s polling
  const statusQuery = useQuery({
    queryKey: UPDATE_STATUS_KEY,
    queryFn: updateService.getStatus,
    refetchInterval: 30000,
    staleTime: 5000,
  });

  // Flow state with adaptive polling (500ms when active, disabled when idle)
  const flowStateQuery = useQuery({
    queryKey: UPDATE_FLOW_STATE_KEY,
    queryFn: updateService.getFlowState,
    refetchInterval: (query) => {
      const state = query.state === 'success' ? query.data?.state : null;
      if (state === 'Idle' || state === 'Completed' || state === 'Failed') {
        return false;
      }
      return 500;
    },
    staleTime: 250,
    retry: 1,
    retryDelay: 100,
  });

  // History query
  const historyQuery = useQuery({
    queryKey: ['updateHistory'],
    queryFn: updateService.getHistory,
  });

  return {
    status: statusQuery.data,
    statusLoading: statusQuery.isLoading,
    statusRefetch: statusQuery.refetch,

    flowState: flowStateQuery.data,
    flowStateLoading: flowStateQuery.isLoading,

    history: historyQuery.data ?? [],
    historyLoading: historyQuery.isLoading,
  };
}
