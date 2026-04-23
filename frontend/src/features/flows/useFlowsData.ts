import { useQuery } from '@tanstack/react-query'
import { api, type FlowList, type OperationStatus } from '../../api'
import { useAuth } from '../auth/useAuth'

export interface UseFlowsDataReturn {
  flows: FlowList | undefined
  loading: boolean
  error: unknown
  operationStatus: OperationStatus | undefined
  operationsLoading: boolean
}

export function useFlowsData(): UseFlowsDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const flowsQuery = useQuery({
    queryKey: ['flows'],
    queryFn: api.flows,
    enabled: !!user,
  })

  // NOTE: React Query deduplicates this query — one request fires regardless of how many features call it
  const operationsQuery = useQuery({
    queryKey: ['operations-status'],
    queryFn: api.operationsStatus,
    enabled: !!user && isAdmin,
    refetchInterval: 2000,
  })

  return {
    flows: flowsQuery.data,
    loading: flowsQuery.isLoading,
    error: flowsQuery.error,
    operationStatus: operationsQuery.data,
    operationsLoading: operationsQuery.isLoading,
  }
}
