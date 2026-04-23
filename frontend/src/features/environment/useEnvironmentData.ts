import { useQuery } from '@tanstack/react-query'
import { api, type ManagedEnvState } from '../../api'
import { useAuth } from '../auth/useAuth'

export interface UseEnvironmentDataReturn {
  state: ManagedEnvState | undefined
  loading: boolean
  error: unknown
}

export function useEnvironmentData(): UseEnvironmentDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const environmentQuery = useQuery({
    queryKey: ['environment'],
    queryFn: api.environment,
    enabled: !!user && isAdmin,
  })

  return {
    state: environmentQuery.data,
    loading: environmentQuery.isLoading,
    error: environmentQuery.error,
  }
}
