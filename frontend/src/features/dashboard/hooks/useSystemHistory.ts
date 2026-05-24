import { useQuery } from '@tanstack/react-query';
import { historyService } from '../services/historyService';
import type { MetricsSnapshot } from '../types/history';

export function useSystemHistory() {
  const { data: response, isLoading, isError } = useQuery({
    queryKey: ['system', 'history'],
    queryFn: () => historyService.getSystemHistory(120),
    refetchInterval: 30000,
    retry: false,
    throwOnError: false,
  });

  const data: MetricsSnapshot[] = response?.data?.data ?? [];

  return { data, isLoading, isError };
}
