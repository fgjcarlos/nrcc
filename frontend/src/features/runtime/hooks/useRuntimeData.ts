import { useQuery } from '@tanstack/react-query';
import { runtimeService } from '@/features/configuration/services';

export function useRuntimeData() {
  const { data, isLoading } = useQuery({
    queryKey: ['runtime', 'status'],
    queryFn: () => runtimeService.getStatus(),
    refetchInterval: 5000,
  });

  const runtime = data?.data?.data;

  return {
    runtime,
    isLoading,
  };
}
