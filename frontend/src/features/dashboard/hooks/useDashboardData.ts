import { useQuery } from '@tanstack/react-query';
import { dashboardService } from '../services';
import type { DashboardContainerStatus, DashboardData } from '../types';
import type { HostStatus } from '@/shared/types';

export function useDashboardData(): DashboardData {
  const { data: dockerData, isLoading: dockerLoading, isError: dockerError } = useQuery({
    queryKey: ['docker', 'status'],
    queryFn: () => dashboardService.getDockerStatus(),
    refetchInterval: 10000,
    retry: false,
    throwOnError: false,
  });

  const { data: systemData } = useQuery({
    queryKey: ['system', 'info'],
    queryFn: () => dashboardService.getSystemInfo(),
    refetchInterval: 10000,
  });

  const { data: configData } = useQuery({
    queryKey: ['config'],
    queryFn: () => dashboardService.getConfig(),
    refetchInterval: 60000,
  });

  const { data: hostData } = useQuery({
    queryKey: ['bootstrap', 'status'],
    queryFn: async () => {
      const response = await dashboardService.getHostStatus();
      return response.data?.data as HostStatus;
    },
    refetchInterval: 30000,
  });

  const { data: backupObservability } = useQuery({
	queryKey: ['backups-observability'],
	queryFn: () => dashboardService.getBackupObservability(),
	refetchInterval: 15000,
  });

  return {
    container: dockerData?.data?.data as DashboardContainerStatus | null | undefined,
    system: systemData?.data?.data,
    config: configData?.data?.data as unknown as Record<string, unknown> | undefined,
    host: hostData,
    backups: backupObservability,
    dockerSuccess: dockerData?.data?.success === true,
    dockerLoading,
    dockerError,
  };
}
