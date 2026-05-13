import { useQuery } from '@tanstack/react-query';
import { logService } from '@/features/logs/services';
import { useState } from 'react';
import type { LogLevel } from '@/shared/types';

export function useLogsData() {
  const [levelFilter, setLevelFilter] = useState<LogLevel[]>(['info', 'warn', 'error']);
  const [isPaused, setIsPaused] = useState(false);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['logs', levelFilter.join(',')],
    queryFn: () => logService.getLogs(100, levelFilter.join(',')),
    refetchInterval: isPaused ? false : 3000,
  });

  const logs = data?.data?.data || [];

  return {
    logs,
    isLoading,
    levelFilter,
    setLevelFilter,
    isPaused,
    setIsPaused,
    refetch,
  };
}
