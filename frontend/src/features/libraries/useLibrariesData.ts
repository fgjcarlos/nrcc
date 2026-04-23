import { useQuery } from '@tanstack/react-query'
import { api, type LibraryList, type OperationStatus } from '../../api'
import { useAuth } from '../auth/useAuth'

export interface UseLibrariesDataReturn {
  libraries: LibraryList | undefined
  loading: boolean
  error: unknown
  operationStatus: OperationStatus | undefined
  operationsLoading: boolean
}

export function useLibrariesData(): UseLibrariesDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const librariesQuery = useQuery({
    queryKey: ['libraries'],
    queryFn: api.libraries,
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
    libraries: librariesQuery.data,
    loading: librariesQuery.isLoading,
    error: librariesQuery.error,
    operationStatus: operationsQuery.data,
    operationsLoading: operationsQuery.isLoading,
  }
}
