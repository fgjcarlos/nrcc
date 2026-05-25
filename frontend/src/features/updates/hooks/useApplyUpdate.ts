import { useMutation, useQueryClient } from '@tanstack/react-query';
import { updateService } from '../services/updateService';

import { queryKeys } from '@/shared/lib/queryKeys';
/**
 * Mutation hook to trigger the apply-update flow.
 * 
 * When called, POSTs to /api/updates/apply, which:
 * - Returns immediately (async backend operation)
 * - Backend transitions state: Idle -> BackingUp -> Applying -> Completed/Failed
 * - Frontend polls /api/updates/state (500ms interval) to track progress
 * 
 * On success, invalidates queryKeys.updates.status and queryKeys.updates.flowState
 * to ensure fresh data is fetched by useUpdateStatus and useUpdateFlowState hooks.
 */
export function useApplyUpdate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: updateService.applyUpdate,
    onSuccess: async () => {
      // Invalidate status and flow state queries to trigger refetch
      // This ensures UI is in sync with new backend state
      await queryClient.invalidateQueries({
        queryKey: queryKeys.updates.status,
      });
      await queryClient.invalidateQueries({
        queryKey: queryKeys.updates.flowState,
      });
    },
  });
}
