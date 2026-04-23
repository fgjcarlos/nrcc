import { useQuery } from '@tanstack/react-query'
import { api } from '../../api'
import { useAuth } from '../auth/useAuth'

export interface UseLogsDataReturn {
  logs: string[]
  loading: boolean
  error: unknown
}

export function useLogsData(): UseLogsDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const runtimeLogsQuery = useQuery({
    queryKey: ['runtime-logs'],
    queryFn: api.runtimeLogs,
    enabled: !!user && isAdmin,
    refetchInterval: 5000,
  })

  return {
    logs: runtimeLogsQuery.data?.lines ?? [],
    loading: runtimeLogsQuery.isLoading,
    error: runtimeLogsQuery.error,
  }
}
