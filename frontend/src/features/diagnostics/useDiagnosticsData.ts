import { useQueryClient, useQuery, useMutation } from '@tanstack/react-query'
import { api, type DoctorReport, type LogEntry, type JobRecord } from '../../api'
import { useAuth } from '../auth/useAuth'
import { useToasts } from '../shell/useToasts'

export interface UseDiagnosticsDataReturn {
  report: DoctorReport | undefined
  reportLoading: boolean
  reportError: unknown
  logs: LogEntry[]
  logsLoading: boolean
  logsError: unknown
  jobs: JobRecord[]
  jobsLoading: boolean
  jobsError: unknown
  exporting: boolean
  onRefreshReport: () => Promise<void>
  onRefreshLogs: () => Promise<void>
  onRefreshJobs: () => Promise<void>
  onExport: () => void
}

export function useDiagnosticsData(): UseDiagnosticsDataReturn {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'
  const queryClient = useQueryClient()
  const { pushToast } = useToasts()

  const reportQuery = useQuery({
    queryKey: ['diagnostics-report'],
    queryFn: api.diagnosticsReport,
    enabled: !!user && isAdmin,
    refetchInterval: 30000,
  })

  const logsQuery = useQuery({
    queryKey: ['diagnostics-logs'],
    queryFn: () => api.diagnosticsLogs({ limit: 100 }),
    enabled: !!user && isAdmin,
    refetchInterval: 10000,
  })

  const jobsQuery = useQuery({
    queryKey: ['diagnostics-jobs'],
    queryFn: () => api.diagnosticsJobs({ limit: 50 }),
    enabled: !!user && isAdmin,
    refetchInterval: 10000,
  })

  const exportMutation = useMutation({
    mutationFn: api.diagnosticsExport,
    onSuccess: (result) => {
      pushToast({
        tone: 'success',
        title: 'Support bundle exported',
        detail: `Bundle saved as ${result.path}`,
      })
    },
    onError: (error) => {
      const msg = error instanceof Error ? error.message : 'The support bundle could not be exported.'
      pushToast({
        tone: 'error',
        title: 'Export failed',
        detail: msg,
      })
    },
  })

  return {
    report: reportQuery.data,
    reportLoading: reportQuery.isLoading,
    reportError: reportQuery.error,
    logs: logsQuery.data?.logs ?? [],
    logsLoading: logsQuery.isLoading,
    logsError: logsQuery.error,
    jobs: jobsQuery.data?.jobs ?? [],
    jobsLoading: jobsQuery.isLoading,
    jobsError: jobsQuery.error,
    exporting: exportMutation.isPending,
    onRefreshReport: async () => {
      await queryClient.invalidateQueries({ queryKey: ['diagnostics-report'] })
    },
    onRefreshLogs: async () => {
      await queryClient.invalidateQueries({ queryKey: ['diagnostics-logs'] })
    },
    onRefreshJobs: async () => {
      await queryClient.invalidateQueries({ queryKey: ['diagnostics-jobs'] })
    },
    onExport: () => exportMutation.mutate(),
  }
}
