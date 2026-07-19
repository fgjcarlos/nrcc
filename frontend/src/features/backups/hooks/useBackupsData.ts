import { useQuery } from '@tanstack/react-query';
import { backupService } from '@/features/backups/services';

import { queryKeys } from '@/shared/lib/queryKeys';
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
    queryKey: queryKeys.backups.config,
    queryFn: backupService.getConfig,
  });

  // Scheduler status query
  const statusQuery = useQuery({
    queryKey: queryKeys.backups.status,
    queryFn: backupService.getStatus,
    refetchInterval: 15000,
  });

  // Observability query
  const observabilityQuery = useQuery({
    queryKey: queryKeys.backups.observability,
    queryFn: backupService.getObservability,
    refetchInterval: 15000,
  });

  // Backup list query
  const backupListQuery = useQuery({
    queryKey: queryKeys.backups.list(page, limit, sort, order),
    queryFn: () => backupService.listPaginated({ page, limit, sort, order }),
  });

  // Backup detail query
  const detailQuery = useQuery({
    queryKey: queryKeys.backups.detail(selectedBackupId),
    queryFn: () => backupService.detail(selectedBackupId!, backups[0] || null),
    enabled: !!selectedBackupId,
  });

  // Storage query
  const storageQuery = useQuery({
    queryKey: queryKeys.backups.storage,
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
    statusError: statusQuery.isError,

    observability: observabilityQuery.data,
    observabilityLoading: observabilityQuery.isLoading,
    observabilityError: observabilityQuery.isError,

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
