import { useQuery } from '@tanstack/react-query'
import { api, type UpdateStatus, type OperationStatus } from '../../api'
import { useAuth } from '../auth/useAuth'

export interface UseUpdatesDataReturn {
  updateStatus: UpdateStatus | undefined
  loading: boolean
  error: unknown
  operationStatus: OperationStatus | undefined
  operationsLoading: boolean
}

export function useUpdatesData(): UseUpdatesDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const updatesQuery = useQuery({
    queryKey: ['updates-status'],
    queryFn: api.updateStatus,
    enabled: !!user && isAdmin,
    refetchInterval: 15000,
  })

  // NOTE: React Query deduplicates this query — one request fires regardless of how many features call it
  const operationsQuery = useQuery({
    queryKey: ['operations-status'],
    queryFn: api.operationsStatus,
    enabled: !!user && isAdmin,
    refetchInterval: 2000,
  })

  return {
    updateStatus: updatesQuery.data,
    loading: updatesQuery.isLoading,
    error: updatesQuery.error,
    operationStatus: operationsQuery.data,
    operationsLoading: operationsQuery.isLoading,
  }
}
