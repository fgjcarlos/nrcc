import { useQuery } from '@tanstack/react-query';
import { updateService } from '@/features/updates/services';
import { useUpdateStatus } from './useUpdateStatus';
import { useUpdateFlowState } from './useUpdateFlowState';

import { queryKeys } from '@/shared/lib/queryKeys';

/**
 * Hook to fetch all update-related queries: status, flow state, and history.
 *
 * Composes useUpdateStatus and useUpdateFlowState rather than re-declaring their
 * queries, so the polling intervals / adaptive-polling thresholds live in one
 * place and cannot drift.
 */
export function useUpdatesData() {
  const statusQuery = useUpdateStatus();
  const flowStateQuery = useUpdateFlowState();

  // History query (no shared hook elsewhere).
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
