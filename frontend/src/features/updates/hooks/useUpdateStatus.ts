import { useQuery } from '@tanstack/react-query';
import { updateService } from '../services/updateService';

export const UPDATE_STATUS_KEY = ['updateStatus'] as const;

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
    queryKey: UPDATE_STATUS_KEY,
    queryFn: () => updateService.getStatus(),
    refetchInterval: 30000, // Poll every 30 seconds
    staleTime: 5000,
  });
}
