import { useQuery } from '@tanstack/react-query';
import { backupService } from '../services';
import type { PaginationParams, PaginatedResponse } from '../types';
import type { BackupSummary } from '../services/backupService';

export function useBackupList(params: PaginationParams) {
  return useQuery({
    queryKey: ['backups', params.page, params.limit, params.sort, params.order],
    queryFn: () => backupService.listPaginated(params),
    refetchInterval: 30000,
  });
}
