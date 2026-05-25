import { useQuery } from '@tanstack/react-query';
import { bootstrapService } from '@/features/bootstrap/services';
import type { HostStatus } from '@/shared/types';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useEnvironment() {
  return useQuery({
    queryKey: queryKeys.bootstrap.status,
    queryFn: async () => {
      const response = await bootstrapService.getStatus();
      return response.data?.data as HostStatus;
    },
    refetchInterval: 30000, // Refresh every 30 seconds
    retry: 2,
  });
}
