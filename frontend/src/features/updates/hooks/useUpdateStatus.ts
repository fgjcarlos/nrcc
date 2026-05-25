import { useQuery } from '@tanstack/react-query';
import { updateService } from '../services/updateService';

import { queryKeys } from '@/shared/lib/queryKeys';

export interface UpdateStatus {
  currentVersion: string;
  latestVersion: string;
  isUpdateAvailable: boolean;
  lastChecked: string;
  isFetching: boolean;
  error?: string;
}

/**
 * Hook for polling update status
 * Refetches every 30 seconds
 */
export function useUpdateStatus() {
  return useQuery({
    queryKey: queryKeys.updates.status,
    queryFn: () => updateService.getStatus(),
    refetchInterval: 30000, // Poll every 30 seconds
    staleTime: 5000,
  });
}
