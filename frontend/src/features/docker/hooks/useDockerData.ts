import { useQuery } from '@tanstack/react-query';
import { dockerService } from '@/features/docker/services';

export function useDockerData() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['docker', 'status'],
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
