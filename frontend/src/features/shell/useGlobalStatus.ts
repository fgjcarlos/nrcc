import { useQuery } from '@tanstack/react-query'
import { api } from '../../api'
import { buildGlobalStatus } from '../../common/utils/status'
import { useAuth } from '../auth/useAuth'
import type { GlobalStatus } from '../../common/types'

export function useGlobalStatus(): GlobalStatus {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const runtimeQuery = useQuery({
    queryKey: ['runtime-status'],
    queryFn: api.runtimeStatus,
    enabled: !!user && isAdmin,
    refetchInterval: 5000,
  })

  const systemInfoQuery = useQuery({
    queryKey: ['system-info'],
    queryFn: api.systemInfo,
    enabled: !!user && isAdmin,
    refetchInterval: 15000,
  })

  return buildGlobalStatus(runtimeQuery.data, runtimeQuery.error, systemInfoQuery.error)
}
