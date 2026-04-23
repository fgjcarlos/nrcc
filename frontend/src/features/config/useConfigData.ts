import { useQuery } from '@tanstack/react-query'
import { api, type OperationStatus } from '../../api'
import { useAuth } from '../auth/useAuth'

export interface UseConfigDataReturn {
  operationStatus: OperationStatus | undefined
  operationsLoading: boolean
}

export function useConfigData(): UseConfigDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  // operationsStatus is polled at 2000ms and deduplicated by React Query.
  // Shared by: Overview, Backups, Flows, Libraries, Updates, Config, Environment.
  // Only 1 HTTP request fires regardless of how many features call this query key.
  const operationsQuery = useQuery({
    queryKey: ['operations-status'],
    queryFn: api.operationsStatus,
    enabled: !!user && isAdmin,
    refetchInterval: 2000,
  })

  return {
    operationStatus: operationsQuery.data,
    operationsLoading: operationsQuery.isLoading,
  }
}
