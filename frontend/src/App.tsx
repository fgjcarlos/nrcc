import { FormEvent, useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { NavLink, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'

import {
  APIRequestError,
  api,
  type BackupList,
  type ConfigValidationResult,
  type DoctorReport,
  type ExportResponse,
  type JobRecord,
  type JobStatus,
  type LibraryList,
  type LogEntry,
  type LogLevel,
  type ManagedEnvState,
  type ManagedEnvVar,
  type OperationStatus,
  type RuntimeStatus,
  type SupportedConfig,
  type SystemInfo,
  type UpdateStatus,
  type User,
} from './api'
import { ConfigPage } from './pages/ConfigPage'

type AuthMode = 'login' | 'register'
type PageKey = 'overview' | 'logs' | 'config' | 'environment' | 'backups' | 'libraries' | 'updates' | 'diagnostics'
type ToastTone = 'success' | 'error' | 'info'

type Toast = {
  id: number
  title: string
  detail?: string
  tone: ToastTone
}

export function App() {
  const queryClient = useQueryClient()
  const location = useLocation()
  const navigate = useNavigate()
  const [authMode, setAuthMode] = useState<AuthMode>('login')
  const [authMessage, setAuthMessage] = useState('')
  const [toasts, setToasts] = useState<Toast[]>([])

  const authStatusQuery = useQuery({
    queryKey: ['auth-status'],
    queryFn: api.authStatus,
    retry: false,
  })

  const meQuery = useQuery({
    queryKey: ['me'],
    queryFn: api.me,
    retry: false,
  })

  useEffect(() => {
    if (authStatusQuery.data?.hasUsers === false) {
      setAuthMode('register')
    } else {
      setAuthMode('login')
    }
  }, [authStatusQuery.data?.hasUsers])

  useEffect(() => {
    if (meQuery.isSuccess && location.pathname === '/login') {
      navigate('/app/overview', { replace: true })
    }
    if (meQuery.isError && location.pathname !== '/login') {
      navigate('/login', { replace: true })
    }
  }, [location.pathname, meQuery.isError, meQuery.isSuccess, navigate])

  function pushToast(toast: Omit<Toast, 'id'>) {
    const id = Date.now() + Math.floor(Math.random() * 1000)
    setToasts((current) => [...current, { ...toast, id }])
  }

  function dismissToast(id: number) {
    setToasts((current) => current.filter((toast) => toast.id !== id))
  }

  const loginMutation = useMutation({
    mutationFn: ({ username, password }: { username: string; password: string }) =>
      api.login(username, password),
    onSuccess: async () => {
      setAuthMessage('')
      pushToast({
        tone: 'success',
        title: 'Signed in',
        detail: 'The local administrator session is active.',
      })
      await queryClient.invalidateQueries({ queryKey: ['me'] })
      await queryClient.invalidateQueries({ queryKey: ['auth-status'] })
    },
    onError: (error) => {
      const message = formatErrorMessage(error, 'Login failed')
      setAuthMessage(message)
      pushToast({
        tone: 'error',
        title: 'Login failed',
        detail: message,
      })
    },
  })

  const registerMutation = useMutation({
    mutationFn: ({ username, password }: { username: string; password: string }) =>
      api.register(username, password),
    onSuccess: async () => {
      setAuthMessage('')
      pushToast({
        tone: 'success',
        title: 'Administrator created',
        detail: 'Bootstrap completed and the local session is ready.',
      })
      await queryClient.invalidateQueries({ queryKey: ['me'] })
      await queryClient.invalidateQueries({ queryKey: ['auth-status'] })
    },
    onError: (error) => {
      const message = formatErrorMessage(error, 'Registration failed')
      setAuthMessage(message)
      pushToast({
        tone: 'error',
        title: 'Bootstrap failed',
        detail: message,
      })
    },
  })

  const logoutMutation = useMutation({
    mutationFn: api.logout,
    onSuccess: async () => {
      pushToast({
        tone: 'info',
        title: 'Signed out',
        detail: 'The local session has been closed.',
      })
      await queryClient.invalidateQueries({ queryKey: ['me'] })
    },
    onError: (error) => {
      pushToast({
        tone: 'error',
        title: 'Sign out failed',
        detail: formatErrorMessage(error, 'Could not sign out'),
      })
    },
  })

  const runtimeQuery = useQuery({
    queryKey: ['runtime-status'],
    queryFn: api.runtimeStatus,
    enabled: meQuery.isSuccess,
    refetchInterval: 5000,
  })

  const runtimeLogsQuery = useQuery({
    queryKey: ['runtime-logs'],
    queryFn: api.runtimeLogs,
    enabled: meQuery.isSuccess,
    refetchInterval: 5000,
  })

  const systemInfoQuery = useQuery({
    queryKey: ['system-info'],
    queryFn: api.systemInfo,
    enabled: meQuery.isSuccess,
    refetchInterval: 15000,
  })

  const environmentQuery = useQuery({
    queryKey: ['environment'],
    queryFn: api.environment,
    enabled: meQuery.isSuccess,
  })

  const backupsQuery = useQuery({
    queryKey: ['backups'],
    queryFn: api.backups,
    enabled: meQuery.isSuccess,
  })

  const librariesQuery = useQuery({
    queryKey: ['libraries'],
    queryFn: api.libraries,
    enabled: meQuery.isSuccess,
  })

  const operationsQuery = useQuery({
    queryKey: ['operations-status'],
    queryFn: api.operationsStatus,
    enabled: meQuery.isSuccess,
    refetchInterval: 2000,
  })

   const updatesQuery = useQuery({
     queryKey: ['updates-status'],
     queryFn: api.updateStatus,
     enabled: meQuery.isSuccess,
     refetchInterval: 15000,
   })

   const diagnosticsReportQuery = useQuery({
     queryKey: ['diagnostics-report'],
     queryFn: api.diagnosticsReport,
     enabled: meQuery.isSuccess,
     refetchInterval: 30000,
   })

   const diagnosticsLogsQuery = useQuery({
     queryKey: ['diagnostics-logs'],
     queryFn: () => api.diagnosticsLogs({ limit: 100 }),
     enabled: meQuery.isSuccess,
     refetchInterval: 10000,
   })

   const diagnosticsJobsQuery = useQuery({
     queryKey: ['diagnostics-jobs'],
     queryFn: () => api.diagnosticsJobs({ limit: 50 }),
     enabled: meQuery.isSuccess,
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
       pushToast({
         tone: 'error',
         title: 'Restart failed',
         detail: formatErrorMessage(error, 'Node-RED could not be restarted'),
       })
     },
   })

   const exportMutation = useMutation({
     mutationFn: api.diagnosticsExport,
     onSuccess: (result) => {
       pushToast({
         tone: 'success',
         title: 'Support bundle exported',
         detail: `Bundle saved as ${result.path} (${formatBytes(result.size)})`,
       })
     },
     onError: (error) => {
       pushToast({
         tone: 'error',
         title: 'Export failed',
         detail: formatErrorMessage(error, 'The support bundle could not be exported.'),
       })
     },
   })

  if (authStatusQuery.isLoading || meQuery.isLoading) {
    return (
      <>
        <LoadingScreen label="Loading local control center" />
        <ToastViewport toasts={toasts} onDismiss={dismissToast} />
      </>
    )
  }

  const user = meQuery.data?.user
  const globalStatus = buildGlobalStatus(runtimeQuery.data, runtimeQuery.error, systemInfoQuery.error)

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
                  setAuthMessage('')
                  if (authMode === 'register') {
                    registerMutation.mutate({ username, password })
                    return
                  }
                  loginMutation.mutate({ username, password })
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

function LoadingScreen({ label }: { label: string }) {
  return (
    <main className="auth-shell">
      <section className="auth-panel loading-panel">
        <p className="eyebrow">NRCC</p>
        <h1>{label}</h1>
      </section>
    </main>
  )
}

function AuthScreen({
  mode,
  message,
  busy,
  onModeChange,
  onSubmit,
}: {
  mode: AuthMode
  message: string
  busy: boolean
  onModeChange: (mode: AuthMode) => void
  onSubmit: (username: string, password: string) => void
}) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onSubmit(username, password)
  }

  const title =
    mode === 'register' ? 'Create the first administrator' : 'Sign in to the local control center'

  return (
    <main className="auth-shell">
      <section className="auth-panel">
        <p className="eyebrow">Node-RED Control Center</p>
        <h1>{title}</h1>
        <p className="auth-copy">
          {mode === 'register'
            ? 'This machine has not been initialized yet. Create the first local administrator account.'
            : 'Use your local administrator account to access runtime controls and diagnostics.'}
        </p>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label>
            <span>Username</span>
            <input value={username} onChange={(event) => setUsername(event.target.value)} required />
          </label>
          <label>
            <span>Password</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </label>

          {message ? <p className="auth-error">{message}</p> : null}

          <button className="primary-button" type="submit" disabled={busy}>
            {busy ? 'Working...' : mode === 'register' ? 'Create account' : 'Sign in'}
          </button>
        </form>

        <div className="auth-toggle">
          <button
            className={mode === 'login' ? 'ghost-button active' : 'ghost-button'}
            type="button"
            onClick={() => onModeChange('login')}
          >
            Login
          </button>
          <button
            className={mode === 'register' ? 'ghost-button active' : 'ghost-button'}
            type="button"
            onClick={() => onModeChange('register')}
          >
            Bootstrap
          </button>
        </div>
      </section>
    </main>
  )
}

function DashboardShell({
  user,
  globalStatus,
  logoutBusy,
  onLogout,
  children,
}: {
  user: User
  globalStatus: GlobalStatus
  logoutBusy: boolean
  onLogout: () => void
  children: React.ReactNode
}) {
  const items: Array<{ to: string; label: string; page: PageKey }> = [
    { to: '/app/overview', label: 'Overview', page: 'overview' },
    { to: '/app/logs', label: 'Logs', page: 'logs' },
    { to: '/app/diagnostics', label: 'Diagnostics', page: 'diagnostics' },
    { to: '/app/config', label: 'Config', page: 'config' },
    { to: '/app/environment', label: 'Environment', page: 'environment' },
    { to: '/app/backups', label: 'Backups', page: 'backups' },
    { to: '/app/libraries', label: 'Libraries', page: 'libraries' },
    { to: '/app/updates', label: 'Updates', page: 'updates' },
  ]

  return (
    <main className="app-shell">
      <aside className="sidebar">
        <div className="sidebar-top">
          <div>
            <p className="eyebrow">NRCC</p>
            <h1>Control Center</h1>
            <p className="sidebar-copy">
              Local-first operations for Node-RED with Go runtime control and cookie-backed sessions.
            </p>
          </div>

          <section className={`status-banner ${globalStatus.tone}`}>
            <div className="status-banner-copy">
              <p className="status-banner-label">System status</p>
              <strong>{globalStatus.title}</strong>
              <p>{globalStatus.detail}</p>
            </div>
          </section>

          <nav className="sidebar-nav" aria-label="Primary">
            {items.map((item) => (
              <NavLink
                key={item.page}
                to={item.to}
                className={({ isActive }) =>
                  isActive ? 'nav-link active' : 'nav-link'
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
        </div>

        <div className="profile-card">
          <p className="profile-name">{user.username}</p>
          <p className="profile-role">{user.role}</p>
          <button className="ghost-button wide" type="button" onClick={onLogout} disabled={logoutBusy}>
            {logoutBusy ? 'Signing out...' : 'Sign out'}
          </button>
        </div>
      </aside>

      <section className="content">{children}</section>
    </main>
  )
}

function OverviewPage({
  runtime,
  runtimeLoading,
  runtimeError,
  systemInfo,
  systemLoading,
  systemError,
  restarting,
  onRestart,
  globalStatus,
}: {
  runtime?: RuntimeStatus
  runtimeLoading: boolean
  runtimeError: unknown
  systemInfo?: SystemInfo
  systemLoading: boolean
  systemError: unknown
  restarting: boolean
  onRestart: () => void
  globalStatus: GlobalStatus
}) {
  const [confirmRestart, setConfirmRestart] = useState(false)
  const pageError = runtimeError ?? systemError

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Dashboard</h2>
        </div>
        <div className="topbar-actions">
          {confirmRestart ? (
            <>
              <button
                className="ghost-button"
                type="button"
                onClick={() => setConfirmRestart(false)}
                disabled={restarting}
              >
                Cancel
              </button>
              <button
                className="primary-button"
                type="button"
                onClick={() => {
                  setConfirmRestart(false)
                  onRestart()
                }}
                disabled={restarting}
              >
                {restarting ? 'Restarting...' : 'Confirm restart'}
              </button>
            </>
          ) : (
            <button className="primary-button" type="button" onClick={() => setConfirmRestart(true)} disabled={restarting}>
              Restart Node-RED
            </button>
          )}
        </div>
      </header>

      {confirmRestart ? (
        <section className="inline-notice warn">
          <strong>Confirm runtime restart</strong>
          <p>Node-RED will be stopped and started again. Status and logs will refresh automatically.</p>
        </section>
      ) : null}

      {pageError ? (
        <section className="inline-notice error">
          <strong>System information is incomplete</strong>
          <p>{formatErrorMessage(pageError, 'The dashboard could not refresh all runtime details.')}</p>
        </section>
      ) : null}

      <section className="stats-grid">
        <StatCard
          label="Runtime state"
          value={runtimeLoading ? 'Loading...' : runtime?.running ? 'Running' : 'Stopped'}
          accent={runtime?.running ? 'ok' : 'warn'}
        />
        <StatCard
          label="Health"
          value={runtimeLoading ? 'Loading...' : runtime?.healthy ? 'Healthy' : 'Unavailable'}
          accent={runtime?.healthy ? 'ok' : 'warn'}
        />
        <StatCard
          label="Version"
          value={runtimeLoading ? 'Loading...' : runtime?.version || 'Unknown'}
          accent="neutral"
        />
        <StatCard label="Global status" value={globalStatus.title} accent={globalStatus.tone} />
      </section>

      <section className="panel-grid">
        <article className="panel">
          <div className="panel-header">
            <h3>Runtime details</h3>
          </div>
          <dl className="details-list">
            <Detail label="PID" value={runtime?.pid ? String(runtime.pid) : 'N/A'} />
            <Detail label="Port" value={runtime?.port ? String(runtime.port) : 'N/A'} />
            <Detail label="Started at" value={runtime?.startedAt || 'N/A'} />
            <Detail label="Uptime" value={runtimeLoading ? 'Loading...' : formatUptime(runtime?.uptimeSec ?? 0)} />
            <Detail label="Data dir" value={runtime?.dataDir || 'N/A'} />
            <Detail label="Last error" value={runtime?.lastError || 'None'} />
          </dl>
        </article>

        <article className="panel">
          <div className="panel-header">
            <h3>System info</h3>
          </div>
          {systemLoading ? (
            <p className="muted">Loading system information...</p>
          ) : (
            <dl className="details-list">
              <Detail label="Hostname" value={systemInfo?.hostname || 'N/A'} />
              <Detail label="OS" value={systemInfo ? `${systemInfo.goos}/${systemInfo.goarch}` : 'N/A'} />
              <Detail label="CPUs" value={systemInfo ? String(systemInfo.cpus) : 'N/A'} />
              <Detail label="Updated" value={systemInfo?.timestamp || 'N/A'} />
            </dl>
          )}
        </article>
      </section>
    </>
  )
}

function LogsPage({
  logs,
  loading,
  error,
}: {
  logs: string[]
  loading: boolean
  error: unknown
}) {
  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Logs</h2>
        </div>
      </header>

      {error ? (
        <section className="inline-notice error">
          <strong>Logs unavailable</strong>
          <p>{formatErrorMessage(error, 'The runtime log stream could not be loaded.')}</p>
        </section>
      ) : null}

      <article className="panel logs-panel">
        <div className="panel-header">
          <h3>Runtime logs</h3>
        </div>
        <div className="log-output">
          {loading ? <p className="muted">Loading logs...</p> : null}
          {!loading && logs.length === 0 ? <p className="muted">No logs captured yet.</p> : null}
          {logs.map((line, index) => (
            <div className="log-line" key={`${index}-${line}`}>
              {line}
            </div>
          ))}
        </div>
      </article>
    </>
  )
}

function EnvironmentPage({
  state,
  loading,
  error,
  onSaved,
  onError,
}: {
  state?: ManagedEnvState
  loading: boolean
  error: unknown
  onSaved: () => Promise<void>
  onError: (message: string) => void
}) {
  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Environment</h2>
        </div>
      </header>

      {error ? (
        <section className="inline-notice error">
          <strong>Environment unavailable</strong>
          <p>{formatErrorMessage(error, 'Managed runtime variables could not be loaded.')}</p>
        </section>
      ) : null}

      <EnvironmentPanel state={state} loading={loading} onSaved={onSaved} onError={onError} />
    </>
  )
}

function BackupsPage({
  backups,
  loading,
  error,
  onChanged,
}: {
  backups?: BackupList
  loading: boolean
  error: unknown
  onChanged: (message: string, tone: ToastTone) => Promise<void>
}) {
  const [restoreTarget, setRestoreTarget] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: api.createBackup,
    onSuccess: async () => {
      await onChanged('A manual backup was created successfully.', 'success')
    },
    onError: async (mutationError) => {
      await onChanged(formatErrorMessage(mutationError, 'The backup could not be created.'), 'error')
    },
  })

  const restoreMutation = useMutation({
    mutationFn: api.restoreBackup,
    onSuccess: async (result) => {
      setRestoreTarget(null)
      await onChanged(
        `Backup restored. Preventive backup created as ${result.preventiveBackupId}.`,
        'success',
      )
    },
    onError: async (mutationError) => {
      setRestoreTarget(null)
      await onChanged(formatErrorMessage(mutationError, 'The backup could not be restored.'), 'error')
    },
  })

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Backups</h2>
        </div>
        <div className="topbar-actions">
          <button
            className="primary-button"
            type="button"
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? 'Creating...' : 'Create backup'}
          </button>
        </div>
      </header>

      {error ? (
        <section className="inline-notice error">
          <strong>Backups unavailable</strong>
          <p>{formatErrorMessage(error, 'Backup history could not be loaded.')}</p>
        </section>
      ) : null}

      <article className="panel">
        <div className="panel-header">
          <h3>Backup history</h3>
        </div>
        {loading ? <p className="muted">Loading backups...</p> : null}
        {!loading && (!backups || backups.items.length === 0) ? <p className="muted">No backups created yet.</p> : null}
        {backups?.items.length ? (
          <div className="backup-list">
            {backups.items.map((backup) => {
              const confirming = restoreTarget === backup.id
              return (
                <article className="backup-card" key={backup.id}>
                  <div className="backup-card-copy">
                    <strong>{backup.id}</strong>
                    <p>{backup.reason}</p>
                    <p>{backup.archiveName}</p>
                    <p>
                      {formatBytes(backup.archiveBytes)} • {backup.createdAt}
                    </p>
                  </div>
                  <div className="backup-card-actions">
                    {confirming ? (
                      <>
                        <button
                          className="ghost-button"
                          type="button"
                          onClick={() => setRestoreTarget(null)}
                          disabled={restoreMutation.isPending}
                        >
                          Cancel
                        </button>
                        <button
                          className="primary-button"
                          type="button"
                          onClick={() => restoreMutation.mutate(backup.id)}
                          disabled={restoreMutation.isPending}
                        >
                          {restoreMutation.isPending ? 'Restoring...' : 'Confirm restore'}
                        </button>
                      </>
                    ) : (
                      <button
                        className="ghost-button"
                        type="button"
                        onClick={() => setRestoreTarget(backup.id)}
                        disabled={restoreMutation.isPending}
                      >
                        Restore
                      </button>
                    )}
                  </div>
                </article>
              )
            })}
          </div>
        ) : null}
      </article>
    </>
  )
}

function LibrariesPage({
  libraries,
  loading,
  error,
  operationStatus,
  onChanged,
}: {
  libraries?: LibraryList
  loading: boolean
  error: unknown
  operationStatus?: OperationStatus
  onChanged: (message: string, tone: ToastTone) => Promise<void>
}) {
  const [packageName, setPackageName] = useState('')

  const installMutation = useMutation({
    mutationFn: api.installLibrary,
    onSuccess: async (result) => {
      setPackageName('')
      await onChanged(result.message, 'success')
    },
    onError: async (mutationError) => {
      await onChanged(formatErrorMessage(mutationError, 'The package could not be installed.'), 'error')
    },
  })

  const uninstallMutation = useMutation({
    mutationFn: api.uninstallLibrary,
    onSuccess: async (result) => {
      await onChanged(result.message, 'success')
    },
    onError: async (mutationError) => {
      await onChanged(formatErrorMessage(mutationError, 'The package could not be removed.'), 'error')
    },
  })

  const busy = operationStatus?.busy ?? false

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Libraries</h2>
        </div>
      </header>

      {error ? (
        <section className="inline-notice error">
          <strong>Libraries unavailable</strong>
          <p>{formatErrorMessage(error, 'Installed packages could not be loaded.')}</p>
        </section>
      ) : null}

      {busy ? (
        <section className="inline-notice warn">
          <strong>System busy</strong>
          <p>
            {operationStatus?.type ? `${operationStatus.type} in progress` : 'Another operation is in progress'}
            {operationStatus?.detail ? `: ${operationStatus.detail}` : '.'}
          </p>
        </section>
      ) : null}

      <article className="panel">
        <div className="panel-header">
          <h3>Install package</h3>
        </div>
        <form
          className="library-form"
          onSubmit={(event) => {
            event.preventDefault()
            installMutation.mutate(packageName)
          }}
        >
          <label>
            <span>Package name</span>
            <input
              value={packageName}
              onChange={(event) => setPackageName(event.target.value)}
              placeholder="@scope/package"
            />
          </label>
          <button
            className="primary-button"
            type="submit"
            disabled={busy || installMutation.isPending || packageName.trim() === ''}
          >
            {installMutation.isPending ? 'Installing...' : 'Install package'}
          </button>
        </form>
      </article>

      <article className="panel">
        <div className="panel-header">
          <h3>Installed packages</h3>
        </div>
        {loading ? <p className="muted">Loading installed packages...</p> : null}
        {!loading && (!libraries || libraries.items.length === 0) ? <p className="muted">No additional packages installed.</p> : null}
        {libraries?.items.length ? (
          <div className="library-list">
            {libraries.items.map((item) => (
              <article className="library-card" key={item.name}>
                <div className="library-card-copy">
                  <strong>{item.name}</strong>
                  <p>{item.version || 'Unknown version'}</p>
                </div>
                <button
                  className="ghost-button"
                  type="button"
                  onClick={() => uninstallMutation.mutate(item.name)}
                  disabled={busy || uninstallMutation.isPending}
                >
                  {uninstallMutation.isPending ? 'Working...' : 'Remove'}
                </button>
              </article>
            ))}
          </div>
        ) : null}
      </article>
    </>
  )
}

function UpdatesPage({
  updateStatus,
  loading,
  error,
  operationStatus,
  onChanged,
}: {
  updateStatus?: UpdateStatus
  loading: boolean
  error: unknown
  operationStatus?: OperationStatus
  onChanged: (message: string, tone: ToastTone) => Promise<void>
}) {
  const [confirmUpdate, setConfirmUpdate] = useState(false)

  const applyMutation = useMutation({
    mutationFn: api.applyUpdate,
    onSuccess: async (result) => {
      setConfirmUpdate(false)
      await onChanged(result.message, result.rolledBack ? 'error' : 'success')
    },
    onError: async (mutationError) => {
      setConfirmUpdate(false)
      await onChanged(formatErrorMessage(mutationError, 'The update could not be applied.'), 'error')
    },
  })

  const busy = operationStatus?.busy ?? false

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Updates</h2>
        </div>
      </header>

      {error ? (
        <section className="inline-notice error">
          <strong>Update status unavailable</strong>
          <p>{formatErrorMessage(error, 'Node-RED update information could not be loaded.')}</p>
        </section>
      ) : null}

      {busy ? (
        <section className="inline-notice warn">
          <strong>System busy</strong>
          <p>
            {operationStatus?.type ? `${operationStatus.type} in progress` : 'Another operation is in progress'}
            {operationStatus?.detail ? `: ${operationStatus.detail}` : '.'}
          </p>
        </section>
      ) : null}

      <article className="panel">
        <div className="panel-header">
          <h3>Node-RED update</h3>
        </div>
        {loading ? <p className="muted">Loading update status...</p> : null}
        {updateStatus ? (
          <div className="update-card">
            <dl className="details-list">
              <Detail label="Installed version" value={updateStatus.installedVersion || 'Unknown'} />
              <Detail label="Available version" value={updateStatus.availableVersion || 'Unknown'} />
              <Detail label="Update available" value={updateStatus.updateAvailable ? 'Yes' : 'No'} />
            </dl>

            {confirmUpdate ? (
              <section className="inline-notice warn">
                <strong>Confirm update</strong>
                <p>A preventive backup will be created before updating Node-RED. Rollback will run automatically if health checks fail.</p>
              </section>
            ) : null}

            <div className="topbar-actions">
              {confirmUpdate ? (
                <>
                  <button
                    className="ghost-button"
                    type="button"
                    onClick={() => setConfirmUpdate(false)}
                    disabled={applyMutation.isPending}
                  >
                    Cancel
                  </button>
                  <button
                    className="primary-button"
                    type="button"
                    onClick={() => applyMutation.mutate()}
                    disabled={busy || applyMutation.isPending}
                  >
                    {applyMutation.isPending ? 'Updating...' : 'Confirm update'}
                  </button>
                </>
              ) : (
                <button
                  className="primary-button"
                  type="button"
                  onClick={() => setConfirmUpdate(true)}
                  disabled={busy || !updateStatus.updateAvailable}
                >
                  {updateStatus.updateAvailable ? 'Update Node-RED' : 'Up to date'}
                </button>
              )}
            </div>
          </div>
         ) : null}
       </article>
     </>
   )
 }

 function DiagnosticsPage({
   report,
   reportLoading,
   reportError,
   logs,
   logsLoading,
   logsError,
   jobs,
   jobsLoading,
   jobsError,
   exporting,
   onRefreshReport,
   onRefreshLogs,
   onRefreshJobs,
   onExport,
 }: {
   report?: DoctorReport
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
 }) {
   const [activeTab, setActiveTab] = useState<'doctor' | 'logs' | 'jobs'>('doctor')

   return (
     <>
       <header className="topbar">
         <div>
           <p className="eyebrow">Support</p>
           <h2>Diagnostics</h2>
         </div>
         <div className="topbar-actions">
           <button
             className="primary-button"
             type="button"
             onClick={onExport}
             disabled={exporting}
           >
             {exporting ? 'Exporting...' : 'Export Support Bundle'}
           </button>
         </div>
       </header>

       <article className="panel diagnostics-panel">
         <div className="panel-header diagnostics-tabs">
           <button
             className={activeTab === 'doctor' ? 'tab-button active' : 'tab-button'}
             type="button"
             onClick={() => setActiveTab('doctor')}
           >
             Doctor
           </button>
           <button
             className={activeTab === 'logs' ? 'tab-button active' : 'tab-button'}
             type="button"
             onClick={() => setActiveTab('logs')}
           >
             Logs
           </button>
           <button
             className={activeTab === 'jobs' ? 'tab-button active' : 'tab-button'}
             type="button"
             onClick={() => setActiveTab('jobs')}
           >
             Jobs
           </button>
         </div>

         {activeTab === 'doctor' && (
           <>
             {reportError ? (
               <section className="inline-notice error">
                 <strong>Doctor report unavailable</strong>
                 <p>{formatErrorMessage(reportError, 'The doctor report could not be loaded.')}</p>
               </section>
             ) : null}
             <div className="tab-content">
               <div className="diagnostics-header">
                 <div className="diagnostics-status">
                   {reportLoading ? (
                     <p className="muted">Loading doctor report...</p>
                   ) : report ? (
                     <>
                       <div className={`status-badge ${getStatusBadgeClass(report.overall_status)}`}>
                         {report.overall_status.toUpperCase()}
                       </div>
                       <p className="muted">Generated at {new Date(report.generated_at).toLocaleString()}</p>
                     </>
                   ) : null}
                 </div>
                 <button
                   className="ghost-button"
                   type="button"
                   onClick={onRefreshReport}
                   disabled={reportLoading}
                 >
                   Refresh
                 </button>
               </div>

               {report && (
                 <div className="checks-list">
                   {report.checks.map((check) => (
                     <div key={check.name} className={`check-item ${check.status}`}>
                       <div className="check-status">
                         <span className="check-icon">
                           {check.status === 'pass' && '✅'}
                           {check.status === 'warn' && '⚠️'}
                           {check.status === 'fail' && '❌'}
                           {check.status === 'unknown' && '❓'}
                         </span>
                         <strong>{formatCheckName(check.name)}</strong>
                       </div>
                       <p>{check.message}</p>
                       {check.details && (
                         <details className="check-details">
                           <summary>Details</summary>
                           <pre>{JSON.stringify(check.details, null, 2)}</pre>
                         </details>
                       )}
                     </div>
                   ))}
                 </div>
               )}
             </div>
           </>
         )}

         {activeTab === 'logs' && (
           <>
             {logsError ? (
               <section className="inline-notice error">
                 <strong>Logs unavailable</strong>
                 <p>{formatErrorMessage(logsError, 'The logs could not be loaded.')}</p>
               </section>
             ) : null}
             <div className="tab-content">
               <div className="diagnostics-header">
                 <p className="muted">{logs.length} logs</p>
                 <button
                   className="ghost-button"
                   type="button"
                   onClick={onRefreshLogs}
                   disabled={logsLoading}
                 >
                   Refresh
                 </button>
               </div>

               <div className="logs-list">
                 {logsLoading ? (
                   <p className="muted">Loading logs...</p>
                 ) : logs.length === 0 ? (
                   <p className="muted">No logs captured yet.</p>
                 ) : (
                   logs.map((log, idx) => (
                     <div key={log.id || idx} className={`log-item level-${log.level}`}>
                       <div className="log-meta">
                         <span className="log-timestamp">
                           {new Date(log.timestamp).toLocaleTimeString()}
                         </span>
                         <span className={`log-badge level-${log.level}`}>
                           {log.level.toUpperCase()}
                         </span>
                         <span className="log-source">{log.source}</span>
                       </div>
                       <div className="log-message">{log.message}</div>
                     </div>
                   ))
                 )}
               </div>
             </div>
           </>
         )}

         {activeTab === 'jobs' && (
           <>
             {jobsError ? (
               <section className="inline-notice error">
                 <strong>Jobs unavailable</strong>
                 <p>{formatErrorMessage(jobsError, 'The jobs history could not be loaded.')}</p>
               </section>
             ) : null}
             <div className="tab-content">
               <div className="diagnostics-header">
                 <p className="muted">{jobs.length} jobs</p>
                 <button
                   className="ghost-button"
                   type="button"
                   onClick={onRefreshJobs}
                   disabled={jobsLoading}
                 >
                   Refresh
                 </button>
               </div>

               <div className="jobs-list">
                 {jobsLoading ? (
                   <p className="muted">Loading jobs...</p>
                 ) : jobs.length === 0 ? (
                   <p className="muted">No jobs recorded yet.</p>
                 ) : (
                   <table className="jobs-table">
                     <thead>
                       <tr>
                         <th>Type</th>
                         <th>Status</th>
                         <th>Started</th>
                         <th>Duration</th>
                         <th>Summary</th>
                       </tr>
                     </thead>
                     <tbody>
                       {jobs.map((job) => (
                         <tr key={job.id} className={`status-${job.status}`}>
                           <td className="job-type">{job.type}</td>
                           <td>
                             <span className={`job-badge status-${job.status}`}>
                               {job.status.toUpperCase()}
                             </span>
                           </td>
                           <td>{new Date(job.started_at).toLocaleString()}</td>
                           <td>
                             {job.finished_at
                               ? formatDuration(
                                   new Date(job.finished_at).getTime() -
                                     new Date(job.started_at).getTime(),
                                 )
                               : '—'}
                           </td>
                           <td>
                             <span className="job-summary">
                               {job.summary || job.error || '—'}
                             </span>
                           </td>
                         </tr>
                       ))}
                     </tbody>
                   </table>
                 )}
               </div>
             </div>
           </>
         )}
       </article>
     </>
   )
 }

function ToastViewport({
  toasts,
  onDismiss,
}: {
  toasts: Toast[]
  onDismiss: (id: number) => void
}) {
  useEffect(() => {
    if (toasts.length === 0) {
      return
    }

    const timers = toasts.map((toast) =>
      window.setTimeout(() => {
        onDismiss(toast.id)
      }, 5000),
    )

    return () => {
      timers.forEach((timer) => window.clearTimeout(timer))
    }
  }, [onDismiss, toasts])

  if (toasts.length === 0) {
    return null
  }

  return (
    <div className="toast-stack" aria-live="polite" aria-atomic="true">
      {toasts.map((toast) => (
        <article key={toast.id} className={`toast ${toast.tone}`}>
          <div>
            <strong>{toast.title}</strong>
            {toast.detail ? <p>{toast.detail}</p> : null}
          </div>
          <button type="button" className="toast-dismiss" onClick={() => onDismiss(toast.id)}>
            Close
          </button>
        </article>
      ))}
    </div>
  )
}

function EnvironmentPanel({
  state,
  loading,
  onSaved,
  onError,
}: {
  state?: ManagedEnvState
  loading: boolean
  onSaved: () => Promise<void>
  onError: (message: string) => void
}) {
  const [variables, setVariables] = useState<ManagedEnvVar[]>([])
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (state) {
      setVariables(state.variables.length > 0 ? state.variables : [{ name: '', value: '' }])
      setMessage('')
    }
  }, [state])

  const applyMutation = useMutation({
    mutationFn: (payload: ManagedEnvVar[]) => api.applyEnvironment(payload),
    onSuccess: async () => {
      setMessage('Managed environment saved. Restart Node-RED to apply the changes.')
      await onSaved()
    },
    onError: (error) => {
      const next = formatErrorMessage(error, 'Save failed')
      setMessage(next)
      onError(next)
    },
  })

  if (loading || !state) {
    return (
      <article className="panel">
        <div className="panel-header">
          <h3>Managed runtime variables</h3>
        </div>
        <p className="muted">Loading managed environment...</p>
      </article>
    )
  }

  function update(index: number, patch: Partial<ManagedEnvVar>) {
    setVariables((current) =>
      current.map((variable, currentIndex) =>
        currentIndex === index ? { ...variable, ...patch } : variable,
      ),
    )
  }

  function addRow() {
    setVariables((current) => [...current, { name: '', value: '' }])
  }

  function removeRow(index: number) {
    setVariables((current) => {
      const next = current.filter((_, currentIndex) => currentIndex !== index)
      return next.length > 0 ? next : [{ name: '', value: '' }]
    })
  }

  return (
    <article className="panel">
      <div className="panel-header">
        <h3>Managed runtime variables</h3>
      </div>
      <p className="muted">
        These variables are injected into the Node-RED runtime from `.env.managed`. Names prefixed with `NRCC_` and `PORT` are reserved.
      </p>

      <form
        className="env-form"
        onSubmit={(event) => {
          event.preventDefault()
          applyMutation.mutate(variables)
        }}
      >
        <div className="env-rows">
          {variables.map((variable, index) => (
            <div className="env-row" key={`${index}-${variable.name}`}>
              <label>
                <span>Name</span>
                <input
                  value={variable.name}
                  onChange={(event) => update(index, { name: event.target.value })}
                  placeholder="API_TOKEN"
                />
              </label>
              <label>
                <span>Value</span>
                <input
                  value={variable.value}
                  onChange={(event) => update(index, { value: event.target.value })}
                  placeholder="secret-value"
                />
              </label>
              <button className="ghost-button env-remove" type="button" onClick={() => removeRow(index)}>
                Remove
              </button>
            </div>
          ))}
        </div>

        <div className="config-actions">
          <button className="ghost-button" type="button" onClick={addRow} disabled={applyMutation.isPending}>
            Add variable
          </button>
          <button className="primary-button" type="submit" disabled={applyMutation.isPending}>
            {applyMutation.isPending ? 'Saving...' : 'Save environment'}
          </button>
        </div>
      </form>

      {message ? <p className="config-message">{message}</p> : null}
    </article>
  )
}

function StatCard({
  label,
  value,
  accent,
}: {
  label: string
  value: string
  accent: 'ok' | 'warn' | 'neutral'
}) {
  return (
    <article className={`stat-card ${accent}`}>
      <p className="stat-label">{label}</p>
      <h3>{value}</h3>
    </article>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="detail-row">
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  )
}

type GlobalStatus = {
  title: string
  detail: string
  tone: 'ok' | 'warn' | 'neutral'
}

function buildGlobalStatus(runtime: RuntimeStatus | undefined, runtimeError: unknown, systemError: unknown): GlobalStatus {
  if (runtimeError || systemError) {
    return {
      title: 'Degraded',
      detail: 'Some dashboard checks failed. Review the active page notices for details.',
      tone: 'warn',
    }
  }

  if (!runtime) {
    return {
      title: 'Unknown',
      detail: 'Waiting for runtime data from the local control center.',
      tone: 'neutral',
    }
  }

  if (runtime.running && runtime.healthy) {
    return {
      title: 'Operational',
      detail: 'Node-RED is running and responding to health checks.',
      tone: 'ok',
    }
  }

  if (runtime.running) {
    return {
      title: 'Needs attention',
      detail: 'Node-RED is running but health checks are not passing yet.',
      tone: 'warn',
    }
  }

  return {
    title: 'Stopped',
    detail: 'Node-RED is not running. Restart the runtime from the dashboard when ready.',
    tone: 'warn',
  }
}

function formatErrorMessage(error: unknown, fallback: string) {
  if (error instanceof APIRequestError) {
    return error.message
  }
  if (error instanceof Error && error.message) {
    return error.message
  }
  return fallback
}

function formatUptime(seconds: number) {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)
  return `${hours}h ${minutes}m ${secs}s`
}

function formatBytes(bytes: number) {
  if (bytes < 1024) {
    return `${bytes} B`
  }
  if (bytes < 1024*1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatCheckName(name: string) {
  return name
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

function getStatusBadgeClass(status: string) {
  switch (status) {
    case 'pass':
      return 'status-pass'
    case 'warn':
      return 'status-warn'
    case 'fail':
      return 'status-fail'
    default:
      return 'status-unknown'
  }
}

function formatDuration(milliseconds: number) {
  const seconds = Math.floor(milliseconds / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`
  }
  return `${seconds}s`
}
