import { useQuery } from '@tanstack/react-query';
import { dockerService } from '@/features/docker/services';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useDockerData() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: queryKeys.docker.status,
    queryFn: () => dockerService.getStatus(),
    refetchInterval: 5000,
  });

  const container = data?.data?.data;

  return {
    container,
    isLoading,
    isError,
    error,
  };
}
