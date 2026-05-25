import { useQuery } from '@tanstack/react-query';
import { backupService } from '../services';
import type { PaginationParams } from '../types';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useBackupList(params: PaginationParams) {
  return useQuery({
    queryKey: queryKeys.backups.legacyList(params.page, params.limit, params.sort, params.order),
    queryFn: () => backupService.listPaginated(params),
    refetchInterval: 30000,
  });
}
