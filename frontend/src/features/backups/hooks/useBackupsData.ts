import { useQuery } from '@tanstack/react-query';
import { backupService } from '@/features/backups/services';

interface UseBackupsDataParams {
  page: number;
  limit: number;
  sort: 'date' | 'size' | 'status';
  order: 'asc' | 'desc';
  selectedBackupId: string | null;
}

export function useBackupsData({
  page,
  limit,
  sort,
  order,
  selectedBackupId,
}: UseBackupsDataParams) {
  // Config query
  const configQuery = useQuery({
    queryKey: ['backups-config'],
    queryFn: backupService.getConfig,
  });

  // Scheduler status query
  const statusQuery = useQuery({
    queryKey: ['backups-status'],
    queryFn: backupService.getStatus,
    refetchInterval: 15000,
  });

  // Observability query
  const observabilityQuery = useQuery({
    queryKey: ['backups-observability'],
    queryFn: backupService.getObservability,
    refetchInterval: 15000,
  });

  // Backup list query
  const backupListQuery = useQuery({
    queryKey: ['backup-list', page, limit, sort, order],
    queryFn: async () => {
      const response = await backupService.getList({
        page,
        limit,
        sort,
        order,
      });
      return response;
    },
  });

  // Backup detail query
  const detailQuery = useQuery({
    queryKey: ['backup-detail', selectedBackupId],
    queryFn: () => backupService.detail(selectedBackupId!, backups[0] || null),
    enabled: !!selectedBackupId,
  });

  // Storage query
  const storageQuery = useQuery({
    queryKey: ['backups-storage'],
    queryFn: backupService.getStorage,
    refetchInterval: 15000,
  });

  // Helper: get current backups list
  const backups = backupListQuery.data?.items ?? [];

  return {
    config: configQuery.data,
    configLoading: configQuery.isLoading,
    configError: configQuery.isError,

    status: statusQuery.data,
    statusLoading: statusQuery.isLoading,

    observability: observabilityQuery.data,
    observabilityLoading: observabilityQuery.isLoading,

    backupList: backupListQuery.data,
    backupsLoading: backupListQuery.isLoading,
    backupsError: backupListQuery.isError,
    backups,

    detail: detailQuery.data,
    detailLoading: detailQuery.isLoading,
    detailFetching: detailQuery.isFetching,

    storage: storageQuery.data,
    storageLoading: storageQuery.isLoading,
    storageError: storageQuery.isError,

    // Refetch functions
    refetchConfig: configQuery.refetch,
    refetchStatus: statusQuery.refetch,
    refetchBackups: backupListQuery.refetch,
  };
}
