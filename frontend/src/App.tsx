import { FormEvent, useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { NavLink, Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'

import {
  APIRequestError,
  api,
  type BackupList,
  type ConfigValidationResult,
  type ManagedEnvState,
  type ManagedEnvVar,
  type RuntimeStatus,
  type SupportedConfig,
  type SystemInfo,
  type User,
} from './api'

type AuthMode = 'login' | 'register'
type PageKey = 'overview' | 'logs' | 'config' | 'environment' | 'backups'
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

  const configQuery = useQuery({
    queryKey: ['config'],
    queryFn: api.config,
    enabled: meQuery.isSuccess,
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
                        config={configQuery.data}
                        configLoading={configQuery.isLoading}
                        configError={configQuery.error}
                        onSaved={async (restartRequired) => {
                          await queryClient.invalidateQueries({ queryKey: ['config'] })
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
    { to: '/app/config', label: 'Config', page: 'config' },
    { to: '/app/environment', label: 'Environment', page: 'environment' },
    { to: '/app/backups', label: 'Backups', page: 'backups' },
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

function ConfigPage({
  config,
  configLoading,
  configError,
  onSaved,
  onError,
}: {
  config?: SupportedConfig
  configLoading: boolean
  configError: unknown
  onSaved: (restartRequired: boolean) => Promise<void>
  onError: (message: string) => void
}) {
  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Config</h2>
        </div>
      </header>

      {configError ? (
        <section className="inline-notice error">
          <strong>Configuration unavailable</strong>
          <p>{formatErrorMessage(configError, 'Supported configuration could not be loaded.')}</p>
        </section>
      ) : null}

      <ConfigPanel config={config} loading={configLoading} onSaved={onSaved} onError={onError} />
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

function ConfigPanel({
  config,
  loading,
  onSaved,
  onError,
}: {
  config?: SupportedConfig
  loading: boolean
  onSaved: (restartRequired: boolean) => Promise<void>
  onError: (message: string) => void
}) {
  const [form, setForm] = useState<SupportedConfig | null>(null)
  const [message, setMessage] = useState('')
  const [validation, setValidation] = useState<ConfigValidationResult | null>(null)

  useEffect(() => {
    if (config) {
      setForm(config)
      setValidation(null)
      setMessage('')
    }
  }, [config])

  const validateMutation = useMutation({
    mutationFn: (payload: SupportedConfig) => api.validateConfig(payload),
    onSuccess: (result) => {
      setValidation(result)
      setMessage(
        result.valid
          ? 'Configuration is valid. A restart will be required.'
          : 'Configuration has validation errors.',
      )
    },
    onError: (error) => {
      const message = formatErrorMessage(error, 'Validation failed')
      setMessage(message)
      onError(message)
    },
  })

  const applyMutation = useMutation({
    mutationFn: (payload: SupportedConfig) => api.applyConfig(payload),
    onSuccess: async (result) => {
      setValidation(result)
      setMessage(result.restartRequired ? 'Configuration saved. Restart Node-RED to apply it.' : 'Configuration saved.')
      await onSaved(result.restartRequired)
    },
    onError: (error) => {
      const message = formatErrorMessage(error, 'Save failed')
      setMessage(message)
      onError(message)
    },
  })

  if (loading || !form) {
    return (
      <article className="panel">
        <div className="panel-header">
          <h3>Supported configuration</h3>
        </div>
        <p className="muted">Loading configuration...</p>
      </article>
    )
  }

  function update<K extends keyof SupportedConfig>(key: K, value: SupportedConfig[K]) {
    setForm((current) => (current ? { ...current, [key]: value } : current))
  }

  return (
    <article className="panel">
      <div className="panel-header">
        <h3>Supported configuration</h3>
      </div>

      <form
        className="config-form"
        onSubmit={(event) => {
          event.preventDefault()
          applyMutation.mutate(form)
        }}
      >
        <label>
          <span>httpAdminRoot</span>
          <input value={form.httpAdminRoot} onChange={(event) => update('httpAdminRoot', event.target.value)} />
        </label>

        <label>
          <span>flowFile</span>
          <input value={form.flowFile} onChange={(event) => update('flowFile', event.target.value)} />
        </label>

        <label>
          <span>credentialSecret</span>
          <input
            type="password"
            value={form.credentialSecret}
            onChange={(event) => update('credentialSecret', event.target.value)}
            placeholder="Leave empty to keep credentialSecret disabled"
          />
        </label>

        <label className="checkbox-row">
          <input
            type="checkbox"
            checked={form.diagnosticsEnabled}
            onChange={(event) => update('diagnosticsEnabled', event.target.checked)}
          />
          <span>Enable diagnostics UI</span>
        </label>

        <label className="checkbox-row">
          <input
            type="checkbox"
            checked={form.projectsEnabled}
            onChange={(event) => update('projectsEnabled', event.target.checked)}
          />
          <span>Enable Node-RED projects</span>
        </label>

        <div className="config-actions">
          <button
            className="ghost-button"
            type="button"
            onClick={() => validateMutation.mutate(form)}
            disabled={validateMutation.isPending || applyMutation.isPending}
          >
            {validateMutation.isPending ? 'Validating...' : 'Validate'}
          </button>
          <button className="primary-button" type="submit" disabled={applyMutation.isPending || validateMutation.isPending}>
            {applyMutation.isPending ? 'Saving...' : 'Save config'}
          </button>
        </div>
      </form>

      {message ? <p className="config-message">{message}</p> : null}

      {validation ? (
        <div className="config-validation">
          {validation.errors.length > 0 ? (
            <ul className="validation-list error-list">
              {validation.errors.map((error) => (
                <li key={error}>{error}</li>
              ))}
            </ul>
          ) : null}

          {validation.diff.length > 0 ? (
            <div>
              <p className="validation-heading">Diff preview</p>
              <ul className="validation-list">
                {validation.diff.map((entry) => (
                  <li key={`${entry.field}-${entry.from}-${entry.to}`}>
                    <strong>{entry.field}</strong>: {entry.from || 'empty'} {'->'} {entry.to || 'empty'}
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </div>
      ) : null}
    </article>
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
