import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { updateService } from '@/features/updates/services';

import { queryKeys } from '@/shared/lib/queryKeys';
import { useT } from '@/i18n';
const DISMISS_KEY = 'cc-update-dismissed-version';

/**
 * Hook for update actions: checking for updates and applying them.
 * Handles mutations and query invalidation on success.
 */
export function useUpdatesActions() {
  const queryClient = useQueryClient();
  const { t } = useT();

  // Check for updates mutation
  const checkMutation = useMutation({
    mutationFn: updateService.check,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.updates.status });
      toast.success(t('updates:checkCompleted'));
    },
    onError: () => {
      toast.error(t('updates:checkFailed'));
    },
  });

  // Apply update mutation
  const applyMutation = useMutation({
    mutationFn: updateService.applyUpdate,
    onSuccess: async (data) => {
      if (data.success) {
        const toVersion = data.toVersion || t('updates:latest');
        if (data.toVersion) {
          localStorage.setItem(DISMISS_KEY, data.toVersion);
        }
        toast.success(t('updates:updatedTo', { version: toVersion }));
        await queryClient.invalidateQueries({ queryKey: queryKeys.updates.status });
        await queryClient.invalidateQueries({ queryKey: queryKeys.updates.flowState });
        await queryClient.invalidateQueries({ queryKey: queryKeys.updates.history });
      } else {
        toast.error(data.message || t('updates:applyFailed'));
      }
    },
    onError: () => {
      toast.error(t('updates:applyFailed'));
    },
  });

  return {
    checkMutation,
    applyMutation,
  };
}
