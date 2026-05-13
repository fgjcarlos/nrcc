import { useQuery } from '@tanstack/react-query';
import { updateService } from '../services/updateService';

export const UPDATE_FLOW_STATE_KEY = ['updateFlowState'] as const;

/**
 * Hook to poll the update flow state from the server.
 * 
 * When an update is in progress (state != Idle/Completed/Failed), polls every 500ms.
 * Otherwise, disabled (no polling).
 * 
 * This allows the frontend to track real-time progress: BackingUp -> Applying -> Completed/Failed
 */
export function useUpdateFlowState() {
  return useQuery({
    queryKey: UPDATE_FLOW_STATE_KEY,
    queryFn: updateService.getFlowState,
    // Aggressive polling while update is active; disabled otherwise
    refetchInterval: (query) => {
      const state = query.state === 'success' ? query.data?.state : null;
      // If state is Idle, Completed, or Failed, disable polling (return false)
      if (state === 'Idle' || state === 'Completed' || state === 'Failed') {
        return false;
      }
      // Otherwise poll every 500ms (BackingUp, Applying, Checking states)
      return 500;
    },
    staleTime: 250, // Data is stale after 250ms, so polling fetches fresh state
    retry: 1,
    retryDelay: 100,
  });
}
