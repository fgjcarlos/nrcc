import { useQuery } from '@tanstack/react-query';
import { logService } from '@/features/logs/services';
import { useState } from 'react';
import type { LogLevel } from '@/shared/types';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useLogsData() {
  const [levelFilter, setLevelFilter] = useState<LogLevel[]>(['info', 'warn', 'error']);
  const [isPaused, setIsPaused] = useState(false);

  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: queryKeys.logs.list(levelFilter),
    queryFn: () => logService.getLogs(100, levelFilter.join(',')),
    refetchInterval: isPaused ? false : 3000,
  });

  const logs = data?.data?.data || [];

  return {
    logs,
    isLoading,
    isError,
    error,
    levelFilter,
    setLevelFilter,
    isPaused,
    setIsPaused,
    refetch,
  };
}
