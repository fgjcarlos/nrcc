import { FormEvent, useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import {
  api,
  type ConfigValidationResult,
  type RuntimeStatus,
  type SupportedConfig,
  type SystemInfo,
  type User,
} from './api'

type AuthMode = 'login' | 'register'

export function App() {
  const queryClient = useQueryClient()
  const [authMode, setAuthMode] = useState<AuthMode>('login')
  const [authMessage, setAuthMessage] = useState('')

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

  const loginMutation = useMutation({
    mutationFn: ({ username, password }: { username: string; password: string }) =>
      api.login(username, password),
    onSuccess: async () => {
      setAuthMessage('')
      await queryClient.invalidateQueries({ queryKey: ['me'] })
      await queryClient.invalidateQueries({ queryKey: ['auth-status'] })
    },
    onError: (error) => {
      setAuthMessage(error instanceof Error ? error.message : 'Login failed')
    },
  })

  const registerMutation = useMutation({
    mutationFn: ({ username, password }: { username: string; password: string }) =>
      api.register(username, password),
    onSuccess: async () => {
      setAuthMessage('')
      await queryClient.invalidateQueries({ queryKey: ['me'] })
      await queryClient.invalidateQueries({ queryKey: ['auth-status'] })
    },
    onError: (error) => {
      setAuthMessage(error instanceof Error ? error.message : 'Registration failed')
    },
  })

  const logoutMutation = useMutation({
    mutationFn: api.logout,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['me'] })
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

  const restartMutation = useMutation({
    mutationFn: api.runtimeRestart,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['runtime-status'] })
      await queryClient.invalidateQueries({ queryKey: ['runtime-logs'] })
    },
  })

  if (authStatusQuery.isLoading || meQuery.isLoading) {
    return <LoadingScreen label="Loading local control center" />
  }

  if (meQuery.isError) {
    return (
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

  const user = meQuery.data?.user
  if (!user) {
    return <LoadingScreen label="Restoring session" />
  }

  return (
    <Dashboard
      user={user}
      runtime={runtimeQuery.data}
      runtimeLoading={runtimeQuery.isLoading}
      logs={runtimeLogsQuery.data?.lines ?? []}
      systemInfo={systemInfoQuery.data}
      systemLoading={systemInfoQuery.isLoading}
      config={configQuery.data}
      configLoading={configQuery.isLoading}
      restarting={restartMutation.isPending}
      logoutBusy={logoutMutation.isPending}
      onRestart={() => restartMutation.mutate()}
      onLogout={() => logoutMutation.mutate()}
      onConfigChanged={async () => {
        await queryClient.invalidateQueries({ queryKey: ['config'] })
      }}
    />
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

function Dashboard({
  user,
  runtime,
  runtimeLoading,
  logs,
  systemInfo,
  systemLoading,
  config,
  configLoading,
  restarting,
  logoutBusy,
  onRestart,
  onLogout,
  onConfigChanged,
}: {
  user: User
  runtime?: RuntimeStatus
  runtimeLoading: boolean
  logs: string[]
  systemInfo?: SystemInfo
  systemLoading: boolean
  config?: SupportedConfig
  configLoading: boolean
  restarting: boolean
  logoutBusy: boolean
  onRestart: () => void
  onLogout: () => void
  onConfigChanged: () => Promise<void>
}) {
  return (
    <main className="app-shell">
      <aside className="sidebar">
        <div>
          <p className="eyebrow">NRCC</p>
          <h1>Control Center</h1>
          <p className="sidebar-copy">
            Local-first operations for Node-RED with Go runtime control and cookie-backed sessions.
          </p>
        </div>

        <div className="profile-card">
          <p className="profile-name">{user.username}</p>
          <p className="profile-role">{user.role}</p>
          <button className="ghost-button wide" type="button" onClick={onLogout} disabled={logoutBusy}>
            {logoutBusy ? 'Signing out...' : 'Sign out'}
          </button>
        </div>
      </aside>

      <section className="content">
        <header className="topbar">
          <div>
            <p className="eyebrow">Runtime</p>
            <h2>Dashboard</h2>
          </div>
          <button className="primary-button" type="button" onClick={onRestart} disabled={restarting}>
            {restarting ? 'Restarting...' : 'Restart Node-RED'}
          </button>
        </header>

        <section className="stats-grid">
          <StatCard
            label="Runtime state"
            value={runtimeLoading ? 'Loading...' : runtime?.running ? 'Running' : 'Stopped'}
            accent={runtime?.healthy ? 'ok' : 'warn'}
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
          <StatCard
            label="Uptime"
            value={runtimeLoading ? 'Loading...' : formatUptime(runtime?.uptimeSec ?? 0)}
            accent="neutral"
          />
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
              <Detail label="Data dir" value={runtime?.dataDir || 'N/A'} />
              <Detail label="Binary path" value={runtime?.binaryPath || 'N/A'} />
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

        <ConfigPanel config={config} loading={configLoading} onSaved={onConfigChanged} />

        <article className="panel logs-panel">
          <div className="panel-header">
            <h3>Runtime logs</h3>
          </div>
          <div className="log-output">
            {logs.length === 0 ? <p className="muted">No logs captured yet.</p> : null}
            {logs.map((line) => (
              <div className="log-line" key={line}>
                {line}
              </div>
            ))}
          </div>
        </article>
      </section>
    </main>
  )
}

function ConfigPanel({
  config,
  loading,
  onSaved,
}: {
  config?: SupportedConfig
  loading: boolean
  onSaved: () => Promise<void>
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
      setMessage(result.valid ? 'Configuration is valid. A restart will be required.' : 'Configuration has validation errors.')
    },
    onError: (error) => {
      setMessage(error instanceof Error ? error.message : 'Validation failed')
    },
  })

  const applyMutation = useMutation({
    mutationFn: (payload: SupportedConfig) => api.applyConfig(payload),
    onSuccess: async (result) => {
      setValidation(result)
      setMessage(result.restartRequired ? 'Configuration saved. Restart Node-RED to apply it.' : 'Configuration saved.')
      await onSaved()
    },
    onError: (error) => {
      setMessage(error instanceof Error ? error.message : 'Save failed')
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

function formatUptime(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = Math.floor(totalSeconds % 60)
  return `${hours}h ${minutes}m ${seconds}s`
}
