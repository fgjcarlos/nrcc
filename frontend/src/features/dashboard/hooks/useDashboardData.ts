import { useQuery } from '@tanstack/react-query';
import { dashboardService, historyService } from '../services';
import type { DashboardContainerStatus, DashboardData } from '../types';
import type { HostStatus } from '@/shared/types';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useDashboardData(): DashboardData {
  const { data: dockerData, isLoading: dockerLoading, isError: dockerError } = useQuery({
    queryKey: queryKeys.docker.status,
    queryFn: () => dashboardService.getDockerStatus(),
    refetchInterval: 10000,
    retry: false,
    throwOnError: false,
  });

  const { data: systemData } = useQuery({
    queryKey: queryKeys.system.info,
    queryFn: () => dashboardService.getSystemInfo(),
    refetchInterval: 10000,
  });

  const { data: configData } = useQuery({
    queryKey: queryKeys.config.root,
    queryFn: () => dashboardService.getConfig(),
    refetchInterval: 60000,
  });

  const { data: hostData } = useQuery({
    queryKey: queryKeys.bootstrap.status,
    queryFn: async () => {
      const response = await dashboardService.getHostStatus();
      return response.data?.data as HostStatus;
    },
    refetchInterval: 30000,
  });

  const { data: backupObservability } = useQuery({
    queryKey: queryKeys.backups.observability,
    queryFn: () => dashboardService.getBackupObservability(),
    refetchInterval: 15000,
  });

  const { data: runtimeData } = useQuery({
    queryKey: queryKeys.runtime.status,
    queryFn: async () => {
      const response = await historyService.getRuntimeHistory(1);
      const status = response.data.data?.status;
      if (!status) {
        throw new Error('Runtime status missing from API response');
      }
      return status;
    },
    refetchInterval: 5000,
  });

  return {
    container: dockerData?.data?.data as DashboardContainerStatus | null | undefined,
    system: systemData?.data?.data,
    config: configData?.data?.data as unknown as Record<string, unknown> | undefined,
    host: hostData,
    runtime: runtimeData,
    backups: backupObservability,
    dockerSuccess: dockerData?.data?.success === true,
    dockerLoading,
    dockerError,
  };
}
