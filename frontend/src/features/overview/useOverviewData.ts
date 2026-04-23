import { useQueryClient, useQuery, useMutation } from '@tanstack/react-query'
import { api, type RuntimeStatus, type SystemInfo, type BackupList, type ManagedEnvState, type OperationStatus } from '../../api'
import { buildGlobalStatus } from '../../common/utils/status'
import { useAuth } from '../auth/useAuth'
import { useToasts } from '../shell/useToasts'
import type { GlobalStatus } from '../../common/types'

export interface UseOverviewDataReturn {
  runtime: RuntimeStatus | undefined
  runtimeLoading: boolean
  runtimeError: unknown
  systemInfo: SystemInfo | undefined
  systemLoading: boolean
  systemError: unknown
  backups: BackupList | undefined
  backupsLoading: boolean
  environment: ManagedEnvState | undefined
  environmentLoading: boolean
  operationStatus: OperationStatus | undefined
  globalStatus: GlobalStatus
  restarting: boolean
  onRestart: () => void
}

export function useOverviewData(): UseOverviewDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'
  const queryClient = useQueryClient()
  const { pushToast } = useToasts()

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

  const backupsQuery = useQuery({
    queryKey: ['backups'],
    queryFn: api.backups,
    enabled: !!user && isAdmin,
  })

  const environmentQuery = useQuery({
    queryKey: ['environment'],
    queryFn: api.environment,
    enabled: !!user && isAdmin,
  })

  // NOTE: React Query deduplicates this query — one request fires regardless of how many features call it
  const operationsQuery = useQuery({
    queryKey: ['operations-status'],
    queryFn: api.operationsStatus,
    enabled: !!user && isAdmin,
    refetchInterval: 2000,
  })

  const restartMutation = useMutation({
    mutationFn: api.runtimeRestart,
    onSuccess: async () => {
      pushToast({
        tone: 'success',
        title: 'Restart requested',
        detail: 'Node-RED is restarting and status will refresh automatically.',
      })
      await queryClient.invalidateQueries({ queryKey: ['operations-status'] })
      await queryClient.invalidateQueries({ queryKey: ['runtime-status'] })
      await queryClient.invalidateQueries({ queryKey: ['runtime-logs'] })
    },
    onError: (error) => {
      const msg = error instanceof Error ? error.message : 'Node-RED could not be restarted'
      pushToast({
        tone: 'error',
        title: 'Restart failed',
        detail: msg,
      })
    },
  })

  const globalStatus = buildGlobalStatus(runtimeQuery.data, runtimeQuery.error, systemInfoQuery.error)

  return {
    runtime: runtimeQuery.data,
    runtimeLoading: runtimeQuery.isLoading,
    runtimeError: runtimeQuery.error,
    systemInfo: systemInfoQuery.data,
    systemLoading: systemInfoQuery.isLoading,
    systemError: systemInfoQuery.error,
    backups: backupsQuery.data,
    backupsLoading: backupsQuery.isLoading,
    environment: environmentQuery.data,
    environmentLoading: environmentQuery.isLoading,
    operationStatus: operationsQuery.data,
    globalStatus,
    restarting: restartMutation.isPending,
    onRestart: () => restartMutation.mutate(),
  }
}
