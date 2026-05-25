import { useQuery } from '@tanstack/react-query';
import { updateService } from '@/features/updates/services';

import { queryKeys } from '@/shared/lib/queryKeys';
/**
 * Hook to fetch all update-related queries: status, flow state, and history.
 * Combines polling for real-time status and history tracking.
 */
export function useUpdatesData() {
  // Status query with 30s polling
  const statusQuery = useQuery({
    queryKey: queryKeys.updates.status,
    queryFn: updateService.getStatus,
    refetchInterval: 30000,
    staleTime: 5000,
  });

  // Flow state with adaptive polling (500ms when active, disabled when idle)
  const flowStateQuery = useQuery({
    queryKey: queryKeys.updates.flowState,
    queryFn: updateService.getFlowState,
    refetchInterval: (query) => {
      // See useUpdateFlowState — `query.state` is an object whose `.status`
      // field holds the fetch status, and `.data` holds the response.
      const state = query.state.status === 'success' ? query.state.data?.state : null;
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
    queryKey: queryKeys.updates.history,
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
