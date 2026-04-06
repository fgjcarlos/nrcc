export type ApiSuccess<T> = {
  success: true
  data: T
  timestamp: string
}

export type ApiFailure = {
  success: false
  error: {
    code: string
    message: string
  }
  timestamp: string
}

export type ApiResponse<T> = ApiSuccess<T> | ApiFailure

export type User = {
  id: string
  username: string
  role: string
  createdAt: string
}

export type AuthSession = {
  user: User
  csrfToken: string
}

export type RuntimeStatus = {
  running: boolean
  healthy: boolean
  pid: number
  port: number
  uptimeSec: number
  version?: string
  dataDir: string
  lastError?: string
  lastExit?: string
  startedAt?: string
  binaryPath?: string
}

export type SystemInfo = {
  goos: string
  goarch: string
  cpus: number
  hostname: string
  timestamp: string
}

export type SupportedConfig = {
  httpAdminRoot: string
  flowFile: string
  diagnosticsEnabled: boolean
  projectsEnabled: boolean
  credentialSecret: string
}

export type ConfigDiffEntry = {
  field: string
  from: string
  to: string
}

export type ConfigValidationResult = {
  valid: boolean
  restartRequired: boolean
  errors: string[]
  diff: ConfigDiffEntry[]
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const method = (init?.method ?? 'GET').toUpperCase()
  const headers = new Headers(init?.headers ?? {})
  headers.set('Content-Type', 'application/json')

  if (requiresCSRF(method) && csrfToken) {
    headers.set('X-CSRF-Token', csrfToken)
  }

  const response = await fetch(path, {
    credentials: 'include',
    headers,
    ...init,
  })

  const payload = (await response.json()) as ApiResponse<T>
  if (!response.ok || !payload.success) {
    const message = payload.success ? 'Request failed' : payload.error.message
    throw new Error(message)
  }

  syncCSRFToken(path, payload.data)
  return payload.data
}

let csrfToken = ''

function syncCSRFToken(path: string, data: unknown) {
  if (path === '/api/auth/logout') {
    csrfToken = ''
    return
  }

  if (!data || typeof data !== 'object' || !('csrfToken' in data)) {
    return
  }

  const nextToken = data.csrfToken
  if (typeof nextToken === 'string') {
    csrfToken = nextToken
  }
}

function requiresCSRF(method: string) {
  return method === 'POST' || method === 'PUT' || method === 'PATCH' || method === 'DELETE'
}

export const api = {
  authStatus: () => request<{ hasUsers: boolean }>('/api/auth/status'),
  me: () => request<AuthSession>('/api/auth/me'),
  login: (username: string, password: string) =>
    request<AuthSession>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  register: (username: string, password: string) =>
    request<AuthSession>('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  logout: () =>
    request<{ loggedOut: boolean }>('/api/auth/logout', {
      method: 'POST',
    }),
  runtimeStatus: () => request<RuntimeStatus>('/api/runtime/status'),
  runtimeLogs: () => request<{ lines: string[] }>('/api/runtime/logs'),
  runtimeRestart: () =>
    request<RuntimeStatus>('/api/runtime/restart', {
      method: 'POST',
    }),
  systemInfo: () => request<SystemInfo>('/api/system/info'),
  config: () => request<SupportedConfig>('/api/config'),
  validateConfig: (config: SupportedConfig) =>
    request<ConfigValidationResult>('/api/config/validate', {
      method: 'POST',
      body: JSON.stringify(config),
    }),
  applyConfig: (config: SupportedConfig) =>
    request<ConfigValidationResult>('/api/config/apply', {
      method: 'POST',
      body: JSON.stringify(config),
    }),
}
