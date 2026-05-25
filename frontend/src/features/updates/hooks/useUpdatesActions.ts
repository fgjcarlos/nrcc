import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { updateService } from '@/features/updates/services';

import { queryKeys } from '@/shared/lib/queryKeys';
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
      await queryClient.invalidateQueries({ queryKey: queryKeys.updates.status });
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
        await queryClient.invalidateQueries({ queryKey: queryKeys.updates.status });
        await queryClient.invalidateQueries({ queryKey: queryKeys.updates.flowState });
        await queryClient.invalidateQueries({ queryKey: queryKeys.updates.history });
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
