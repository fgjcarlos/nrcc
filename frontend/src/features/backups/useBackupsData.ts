import { useQuery } from '@tanstack/react-query'
import { api, type BackupList, type OperationStatus } from '../../api'
import { useAuth } from '../auth/useAuth'

export interface UseBackupsDataReturn {
  backups: BackupList | undefined
  loading: boolean
  error: unknown
  operationStatus: OperationStatus | undefined
  operationsLoading: boolean
}

export function useBackupsData(): UseBackupsDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const backupsQuery = useQuery({
    queryKey: ['backups'],
    queryFn: api.backups,
    enabled: !!user && isAdmin,
  })

  // NOTE: React Query deduplicates this query — one request fires regardless of how many features call it
  const operationsQuery = useQuery({
    queryKey: ['operations-status'],
    queryFn: api.operationsStatus,
    enabled: !!user && isAdmin,
    refetchInterval: 2000,
  })

  return {
    backups: backupsQuery.data,
    loading: backupsQuery.isLoading,
    error: backupsQuery.error,
    operationStatus: operationsQuery.data,
    operationsLoading: operationsQuery.isLoading,
  }
}
