import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { updateService } from '@/features/updates/services';
import { UPDATE_STATUS_KEY } from './useUpdateStatus';
import { UPDATE_FLOW_STATE_KEY } from './useUpdateFlowState';

const DISMISS_KEY = 'cc-update-dismissed-version';

/**
 * Hook for update actions: checking for updates and applying them.
 * Handles mutations and query invalidation on success.
 */
export function useUpdatesActions() {
  const queryClient = useQueryClient();

  // Check for updates mutation
  const checkMutation = useMutation({
    mutationFn: updateService.check,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: UPDATE_STATUS_KEY });
      toast.success('Check completed');
    },
    onError: () => {
      toast.error('Error checking for updates');
    },
  });

  // Apply update mutation
  const applyMutation = useMutation({
    mutationFn: updateService.applyUpdate,
    onSuccess: async (data) => {
      if (data.success) {
        const toVersion = data.toVersion || 'latest';
        if (data.toVersion) {
          localStorage.setItem(DISMISS_KEY, data.toVersion);
        }
        toast.success(`Node-RED updated to ${toVersion}`);
        await queryClient.invalidateQueries({ queryKey: UPDATE_STATUS_KEY });
        await queryClient.invalidateQueries({ queryKey: UPDATE_FLOW_STATE_KEY });
        await queryClient.invalidateQueries({ queryKey: ['updateHistory'] });
      } else {
        toast.error(data.message);
      }
    },
    onError: () => {
      toast.error('Error applying update');
    },
  });

  return {
    checkMutation,
    applyMutation,
  };
}
