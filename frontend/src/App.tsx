import { useEffect } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { api } from './api'
import { buildGlobalStatus } from './common/utils/status'
import { ThemeProvider } from './common/components'
import { useAuth } from './features/auth/useAuth'
import { AuthScreen } from './features/auth/AuthScreen'
import { LoadingScreen } from './features/auth/LoadingScreen'
import { useToasts } from './features/shell/useToasts'
import { DashboardShell } from './features/shell/DashboardShell'
import { ToastViewport } from './features/shell/ToastViewport'
import { OverviewPage } from './features/overview/OverviewPage'
import { LogsPage } from './features/logs/LogsPage'
import { EnvironmentPage } from './features/environment/EnvironmentPage'
import { BackupsPage } from './features/backups/BackupsPage'
import { LibrariesPage } from './features/libraries/LibrariesPage'
import { UpdatesPage } from './features/updates/UpdatesPage'
import { DiagnosticsPage } from './features/diagnostics/DiagnosticsPage'
import { ConfigPage } from './features/config/ConfigPage'

function AppContent() {
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()
  const { toasts, pushToast, dismissToast } = useToasts()
  const { user, isLoading, authMode, setAuthMode, authMessage, loginMutation, registerMutation, logoutMutation } = useAuth()

  // Set up auth navigation effects
  useEffect(() => {
    if (user && location.pathname === '/login') {
      navigate('/app/overview', { replace: true })
    }
    if (!user && location.pathname.startsWith('/app')) {
      navigate('/login', { replace: true })
    }
  }, [user, location.pathname, navigate])

  // Pass pushToast callback to auth mutations for toast notifications
  useEffect(() => {
    if (loginMutation.isSuccess) {
      pushToast({
        tone: 'success',
        title: 'Signed in',
        detail: 'The local administrator session is active.',
      })
    }
  }, [loginMutation.isSuccess])

  useEffect(() => {
    if (registerMutation.isSuccess) {
      pushToast({
        tone: 'success',
        title: 'Administrator created',
        detail: 'Bootstrap completed and the local session is ready.',
      })
    }
  }, [registerMutation.isSuccess])

  useEffect(() => {
    if (logoutMutation.isSuccess) {
      pushToast({
        tone: 'info',
        title: 'Signed out',
        detail: 'The local session has been closed.',
      })
    }
  }, [logoutMutation.isSuccess])

  // All queries enabled only when user is logged in
  const runtimeQuery = useQuery({
    queryKey: ['runtime-status'],
    queryFn: api.runtimeStatus,
    enabled: !!user,
    refetchInterval: 5000,
  })

  const runtimeLogsQuery = useQuery({
    queryKey: ['runtime-logs'],
    queryFn: api.runtimeLogs,
    enabled: !!user,
    refetchInterval: 5000,
  })

  const systemInfoQuery = useQuery({
    queryKey: ['system-info'],
    queryFn: api.systemInfo,
    enabled: !!user,
    refetchInterval: 15000,
  })

  const environmentQuery = useQuery({
    queryKey: ['environment'],
    queryFn: api.environment,
    enabled: !!user,
  })

  const backupsQuery = useQuery({
    queryKey: ['backups'],
    queryFn: api.backups,
    enabled: !!user,
  })

  const librariesQuery = useQuery({
    queryKey: ['libraries'],
    queryFn: api.libraries,
    enabled: !!user,
  })

  const operationsQuery = useQuery({
    queryKey: ['operations-status'],
    queryFn: api.operationsStatus,
    enabled: !!user,
    refetchInterval: 2000,
  })

  const updatesQuery = useQuery({
    queryKey: ['updates-status'],
    queryFn: api.updateStatus,
    enabled: !!user,
    refetchInterval: 15000,
  })

  const diagnosticsReportQuery = useQuery({
    queryKey: ['diagnostics-report'],
    queryFn: api.diagnosticsReport,
    enabled: !!user,
    refetchInterval: 30000,
  })

  const diagnosticsLogsQuery = useQuery({
    queryKey: ['diagnostics-logs'],
    queryFn: () => api.diagnosticsLogs({ limit: 100 }),
    enabled: !!user,
    refetchInterval: 10000,
  })

  const diagnosticsJobsQuery = useQuery({
    queryKey: ['diagnostics-jobs'],
    queryFn: () => api.diagnosticsJobs({ limit: 50 }),
    enabled: !!user,
    refetchInterval: 10000,
  })

  const restartMutation = useMutation({
    mutationFn: api.runtimeRestart,
    onSuccess: async () => {
      pushToast({
        tone: 'success',
        title: 'Restart requested',
        detail: 'Node-RED is restarting and status will refresh automatically.',
      })
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

  const globalStatus = buildGlobalStatus(runtimeQuery.data, runtimeQuery.error, systemInfoQuery.error)

  if (isLoading) {
    return (
      <>
        <LoadingScreen label="Loading local control center" />
        <ToastViewport toasts={toasts} onDismiss={dismissToast} />
      </>
    )
  }

  return (
    <>
      <Routes>
        <Route
          path="/login"
          element={
            user ? (
              <Navigate to="/app/overview" replace />
            ) : (
              <AuthScreen
                mode={authMode}
                message={authMessage}
                busy={loginMutation.isPending || registerMutation.isPending}
                onModeChange={setAuthMode}
                onSubmit={(username, password) => {
                  if (authMode === 'register') {
                    registerMutation.mutate({ username, password })
                  } else {
                    loginMutation.mutate({ username, password })
                  }
                }}
              />
            )
          }
        />
        <Route
          path="/app/*"
          element={
            user ? (
              <DashboardShell
                user={user}
                globalStatus={globalStatus}
                logoutBusy={logoutMutation.isPending}
                onLogout={() => logoutMutation.mutate()}
              >
                <Routes>
                  <Route
                    path="overview"
                    element={
                      <OverviewPage
                        runtime={runtimeQuery.data}
                        runtimeLoading={runtimeQuery.isLoading}
                        runtimeError={runtimeQuery.error}
                        systemInfo={systemInfoQuery.data}
                        systemLoading={systemInfoQuery.isLoading}
                        systemError={systemInfoQuery.error}
                        restarting={restartMutation.isPending}
                        onRestart={() => restartMutation.mutate()}
                        globalStatus={globalStatus}
                      />
                    }
                  />
                  <Route
                    path="logs"
                    element={
                      <LogsPage
                        logs={runtimeLogsQuery.data?.lines ?? []}
                        loading={runtimeLogsQuery.isLoading}
                        error={runtimeLogsQuery.error}
                      />
                    }
                  />
                  <Route
                    path="config"
                    element={
                      <ConfigPage
                        onSaved={(restartRequired) => {
                          pushToast({
                            tone: 'success',
                            title: 'Configuration saved',
                            detail: restartRequired
                              ? 'Saved successfully. Restart Node-RED to apply the changes.'
                              : 'Saved successfully.',
                          })
                        }}
                        onError={(message) => {
                          pushToast({
                            tone: 'error',
                            title: 'Configuration failed',
                            detail: message,
                          })
                        }}
                        onToast={(message, type) => {
                          pushToast({
                            tone: type,
                            title: message.split('\n')[0],
                            detail: message.split('\n').slice(1).join('\n') || undefined,
                          })
                        }}
                      />
                    }
                  />
                  <Route
                    path="environment"
                    element={
                      <EnvironmentPage
                        state={environmentQuery.data}
                        loading={environmentQuery.isLoading}
                        error={environmentQuery.error}
                        onSaved={async () => {
                          await queryClient.invalidateQueries({ queryKey: ['environment'] })
                          pushToast({
                            tone: 'success',
                            title: 'Environment saved',
                            detail: 'Managed runtime variables were updated. Restart Node-RED to apply them.',
                          })
                        }}
                        onError={(message) => {
                          pushToast({
                            tone: 'error',
                            title: 'Environment update failed',
                            detail: message,
                          })
                        }}
                      />
                    }
                  />
                  <Route
                    path="backups"
                    element={
                      <BackupsPage
                        backups={backupsQuery.data}
                        loading={backupsQuery.isLoading}
                        error={backupsQuery.error}
                        onChanged={async (message, tone) => {
                          await queryClient.invalidateQueries({ queryKey: ['backups'] })
                          pushToast({
                            tone,
                            title: tone === 'success' ? 'Backups updated' : 'Backup action failed',
                            detail: message,
                          })
                        }}
                      />
                    }
                  />
                  <Route
                    path="libraries"
                    element={
                      <LibrariesPage
                        libraries={librariesQuery.data}
                        loading={librariesQuery.isLoading}
                        error={librariesQuery.error}
                        operationStatus={operationsQuery.data}
                        onChanged={async (message, tone) => {
                          await queryClient.invalidateQueries({ queryKey: ['libraries'] })
                          await queryClient.invalidateQueries({ queryKey: ['operations-status'] })
                          pushToast({
                            tone,
                            title: tone === 'success' ? 'Libraries updated' : 'Library action failed',
                            detail: message,
                          })
                        }}
                      />
                    }
                  />
                  <Route
                    path="updates"
                    element={
                      <UpdatesPage
                        updateStatus={updatesQuery.data}
                        loading={updatesQuery.isLoading}
                        error={updatesQuery.error}
                        operationStatus={operationsQuery.data}
                        onChanged={async (message, tone) => {
                          await queryClient.invalidateQueries({ queryKey: ['updates-status'] })
                          await queryClient.invalidateQueries({ queryKey: ['operations-status'] })
                          await queryClient.invalidateQueries({ queryKey: ['runtime-status'] })
                          pushToast({
                            tone,
                            title: tone === 'success' ? 'Update completed' : 'Update failed',
                            detail: message,
                          })
                        }}
                      />
                    }
                  />
                  <Route
                    path="diagnostics"
                    element={
                      <DiagnosticsPage
                        report={diagnosticsReportQuery.data}
                        reportLoading={diagnosticsReportQuery.isLoading}
                        reportError={diagnosticsReportQuery.error}
                        logs={diagnosticsLogsQuery.data?.logs ?? []}
                        logsLoading={diagnosticsLogsQuery.isLoading}
                        logsError={diagnosticsLogsQuery.error}
                        jobs={diagnosticsJobsQuery.data?.jobs ?? []}
                        jobsLoading={diagnosticsJobsQuery.isLoading}
                        jobsError={diagnosticsJobsQuery.error}
                        exporting={exportMutation.isPending}
                        onRefreshReport={async () => {
                          await queryClient.invalidateQueries({ queryKey: ['diagnostics-report'] })
                        }}
                        onRefreshLogs={async () => {
                          await queryClient.invalidateQueries({ queryKey: ['diagnostics-logs'] })
                        }}
                        onRefreshJobs={async () => {
                          await queryClient.invalidateQueries({ queryKey: ['diagnostics-jobs'] })
                        }}
                        onExport={() => exportMutation.mutate()}
                      />
                    }
                  />
                  <Route path="*" element={<Navigate to="/app/overview" replace />} />
                </Routes>
              </DashboardShell>
            ) : (
              <Navigate to="/login" replace />
            )
          }
        />
        <Route path="*" element={<Navigate to={user ? '/app/overview' : '/login'} replace />} />
      </Routes>
      <ToastViewport toasts={toasts} onDismiss={dismissToast} />
    </>
  )
}

export function App() {
  return (
    <ThemeProvider>
      <AppContent />
    </ThemeProvider>
  )
}
