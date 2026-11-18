import { useEffect, useRef } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
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
  const queryClient = useQueryClient();

  // When a running update settles, the installed version and history changed.
  const flow = flowStateQuery.data?.state;
  const previousFlow = useRef(flow);
  useEffect(() => {
    const wasRunning = previousFlow.current === 'BackingUp' || previousFlow.current === 'Applying';
    previousFlow.current = flow;
    if (wasRunning && (flow === 'Completed' || flow === 'Failed')) {
      void queryClient.invalidateQueries({ queryKey: queryKeys.updates.status });
      void queryClient.invalidateQueries({ queryKey: queryKeys.updates.history });
    }
  }, [flow, queryClient]);

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
