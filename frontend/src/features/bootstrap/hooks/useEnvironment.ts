import { useQuery } from '@tanstack/react-query';
import { bootstrapService } from '@/features/bootstrap/services';
import type { HostStatusWithCodes } from '@/features/bootstrap/types';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useEnvironment() {
  return useQuery({
    queryKey: queryKeys.bootstrap.status,
    queryFn: async () => {
      const response = await bootstrapService.getStatus();
      return response.data?.data as HostStatusWithCodes;
    },
    refetchInterval: 30000, // Refresh every 30 seconds
    retry: 2,
  });
}
