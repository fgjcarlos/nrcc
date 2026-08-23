import { useQuery } from '@tanstack/react-query';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { queryKeys } from '@/shared/lib/queryKeys';
import { dashboardService } from '../services';
import type { SecurityPosture } from '../types';

export function useSecurityPosture() {
  const { user } = useAuth();
  const query = useQuery({
    queryKey: queryKeys.system.securityPosture,
    queryFn: () => dashboardService.getSecurityPosture(),
    enabled: user?.role === 'admin',
    retry: false,
    throwOnError: false,
  });

  return { ...query, data: query.data?.data?.data as SecurityPosture | undefined };
}
