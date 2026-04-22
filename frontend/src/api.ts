import {
  FullAppConfig,
  ExtendedConfigValidationResult,
  ConfigSnapshot,
  ConfigSnapshotList,
} from './types/config'
import type { ImportResponse } from './common/types'

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
    request_id?: string
    details?: unknown
  }
  request_id?: string
  timestamp: string
}

export type ApiResponse<T> = ApiSuccess<T> | ApiFailure

export class APIRequestError extends Error {
  code?: string
  status: number
  requestId?: string
  details?: unknown

  constructor(message: string, status: number, code?: string, requestId?: string, details?: unknown) {
    super(message)
    this.name = 'APIRequestError'
    this.status = status
    this.code = code
    this.requestId = requestId
    this.details = details
  }
}

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
  localAccess: {
    mode: 'direct' | 'portless'
    hostname?: string
    url: string
    fallbackUrl: string
    portlessAvailable: boolean
    configured: boolean
    operational: boolean
    message: string
  }
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

export type ManagedEnvVar = {
  name: string
  value: string
  secret?: boolean
  hasValue?: boolean
}

export type ManagedEnvState = {
  variables: ManagedEnvVar[]
  restartRequired: boolean
}

export type BackupSummary = {
  id: string
  reason: string
  createdAt: string
  archiveName: string
  archiveBytes: number
  archiveSha256: string
}

export type BackupList = {
  items: BackupSummary[]
}

export type LibraryPackage = {
  name: string
  version: string
  direct: boolean
}

export type LibraryList = {
  items: LibraryPackage[]
}

export type LibraryOperationResult = {
  package: LibraryPackage
  message: string
  output?: string
  operation: string
}

export type FlowSource = {
  userDir: string
  flowFile: string
  path: string
  readOnly: boolean
  updatedAt?: string
}

export type FlowSummary = {
  id: string
  label: string
  nodeCount: number
  disabledNodeCount: number
  customNodeCount: number
  inboundWireCount: number
  outboundWireCount: number
  subflowUsageCount: number
}

export type FlowSummaryTotals = {
  flowCount: number
  nodeCount: number
  disabledNodeCount: number
  customNodeCount: number
  inboundWireCount: number
  outboundWireCount: number
  subflowUsageCount: number
}

export type FlowList = {
  source: FlowSource
  summary: FlowSummaryTotals
  items: FlowSummary[]
}

export type FlowTypeMetric = {
  type: string
  count: number
  custom: boolean
}

export type FlowNodeSummary = {
  id: string
  type: string
  name: string
  disabled: boolean
  wireCount: number
}

export type FlowDetail = FlowSummary & {
  nodeTypes: FlowTypeMetric[]
  nodes: FlowNodeSummary[]
}

export type FlowDetailResponse = {
  source: FlowSource
  flow: FlowDetail
}

export type FlowAnalysisProvider = {
  name: string
  model: string
  local: boolean
}

export type FlowAnalysis = {
  source: FlowSource
  flow: FlowSummary
  advisory: boolean
  summary: string
  strengths: string[]
  issues: string[]
  suggestions: string[]
  provider: FlowAnalysisProvider
}

export type OperationStatus = {
  busy: boolean
  type?: string
  detail?: string
  startedAt?: string
}

export type UpdateStatus = {
  installedVersion: string
  availableVersion: string
  updateAvailable: boolean
}

export type UpdateApplyResult = {
  fromVersion: string
  toVersion: string
  preventiveBackupId: string
  rolledBack: boolean
  message: string
}

export type DoctorCheckStatus = 'pass' | 'warn' | 'fail'

export type DoctorCheck = {
  id: string
  label: string
  status: DoctorCheckStatus
  severity?: 'critical' | 'warning'
  message: string
}

export type DoctorReport = {
  generatedAt: string
  overallStatus: 'healthy' | 'degraded' | 'critical'
  checks: DoctorCheck[]
}

export type LogLevel = 'debug' | 'info' | 'warn' | 'error'

export type LogEntry = {
  id?: string
  timestamp: string
  level: LogLevel
  source: string
  event?: string
  message: string
  metadata?: Record<string, unknown>
}

export type LogsResponse = {
  logs: LogEntry[]
  total: number
}

export type JobStatus = 'pending' | 'running' | 'completed' | 'failed'

export type JobRecord = {
  id: string
  type: string
  status: JobStatus
  started_at: string
  finished_at?: string
  triggered_by?: string
  summary?: string
  error?: string
}

export type JobsResponse = {
  jobs: JobRecord[]
  total: number
}

export type ExportResponse = {
  path: string
  size: number
}

export type AssetInfo = {
  id: string
  category: string
  filename: string
  original: string
  mimeType: string
  size: number
  url: string
  createdAt: string
}

export type AssetList = {
  items: AssetInfo[]
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

  let payload: ApiResponse<T> | null = null
  try {
    payload = (await response.json()) as ApiResponse<T>
  } catch {
    payload = null
  }

  if (!response.ok || !payload || !payload.success) {
    const message = payload && !payload.success ? payload.error.message : `Request failed with status ${response.status}`
    const code = payload && !payload.success ? payload.error.code : undefined
    const requestId = payload && !payload.success ? (payload.error.request_id ?? payload.request_id) : undefined
    const details = payload && !payload.success ? payload.error.details : undefined
    throw new APIRequestError(message, response.status, code, requestId, details)
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

async function requestMultipart<T>(path: string, formData: FormData): Promise<T> {
  const headers = new Headers()
  // Do NOT set Content-Type — browser sets it with boundary for multipart
  if (csrfToken) {
    headers.set('X-CSRF-Token', csrfToken)
  }

  const response = await fetch(path, {
    method: 'POST',
    credentials: 'include',
    headers,
    body: formData,
  })

  let payload: ApiResponse<T> | null = null
  try {
    payload = (await response.json()) as ApiResponse<T>
  } catch {
    payload = null
  }

  if (!response.ok || !payload || !payload.success) {
    const message = payload && !payload.success ? payload.error.message : `Request failed with status ${response.status}`
    const code = payload && !payload.success ? payload.error.code : undefined
    throw new APIRequestError(message, response.status, code)
  }

  return payload.data
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
  environment: () => request<ManagedEnvState>('/api/environment'),
  applyEnvironment: (variables: ManagedEnvVar[]) =>
    request<ManagedEnvState>('/api/environment/apply', {
      method: 'POST',
      body: JSON.stringify({ variables }),
    }),
  backups: () => request<BackupList>('/api/backups'),
  createBackup: () =>
    request<BackupSummary>('/api/backups/create', {
      method: 'POST',
    }),
  restoreBackup: (id: string) =>
    request<{ restoredBackupId: string; preventiveBackupId: string }>(`/api/backups/${id}/restore`, {
      method: 'POST',
    }),
  flows: () => request<FlowList>('/api/flows'),
  flow: (id: string) => request<FlowDetailResponse>(`/api/flows/${encodeURIComponent(id)}`),
  analyzeFlow: (id: string) =>
    request<FlowAnalysis>(`/api/flows/${encodeURIComponent(id)}/analysis`, {
      method: 'POST',
    }),
  exportFlows: async (ids: string[]): Promise<Blob> => {
    const headers = new Headers()
    if (csrfToken) {
      headers.set('X-CSRF-Token', csrfToken)
    }

    const response = await fetch('/api/flows/export', {
      method: 'POST',
      credentials: 'include',
      headers: {
        ...Object.fromEntries(headers),
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ ids }),
    })

    if (!response.ok) {
      let errorMessage = `Export failed with status ${response.status}`
      try {
        const payload = (await response.json()) as ApiResponse<unknown>
        if (!payload.success && payload.error) {
          errorMessage = payload.error.message
        }
      } catch {
        // Ignore JSON parse errors
      }
      throw new APIRequestError(errorMessage, response.status)
    }

    return response.blob()
  },
  importFlows: async (file: File): Promise<ImportResponse> => {
    const formData = new FormData()
    formData.append('file', file)
    return requestMultipart<ImportResponse>('/api/flows/import', formData)
  },
  libraries: () => request<LibraryList>('/api/libraries'),
  installLibrary: (name: string) =>
    request<LibraryOperationResult>(`/api/libraries/${encodeURIComponent(name)}`, {
      method: 'POST',
    }),
  uninstallLibrary: (name: string) =>
    request<LibraryOperationResult>(`/api/libraries/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),
   operationsStatus: () => request<OperationStatus>('/api/operations/status'),
   updateStatus: () => request<UpdateStatus>('/api/updates/status'),
   applyUpdate: () =>
     request<UpdateApplyResult>('/api/updates/apply', {
       method: 'POST',
     }),
   diagnosticsReport: () => request<DoctorReport>('/api/diagnostics/report'),
   diagnosticsLogs: (params?: { level?: LogLevel; source?: string; limit?: number; offset?: number }) => {
     const searchParams = new URLSearchParams()
     if (params?.level) searchParams.set('level', params.level)
     if (params?.source) searchParams.set('source', params.source)
     if (params?.limit) searchParams.set('limit', params.limit.toString())
     if (params?.offset) searchParams.set('offset', params.offset.toString())
     const query = searchParams.toString()
     return request<LogsResponse>(`/api/diagnostics/logs${query ? '?' + query : ''}`)
   },
   diagnosticsJobs: (params?: { type?: string; status?: JobStatus; limit?: number; offset?: number }) => {
     const searchParams = new URLSearchParams()
     if (params?.type) searchParams.set('type', params.type)
     if (params?.status) searchParams.set('status', params.status)
     if (params?.limit) searchParams.set('limit', params.limit.toString())
     if (params?.offset) searchParams.set('offset', params.offset.toString())
     const query = searchParams.toString()
     return request<JobsResponse>(`/api/diagnostics/jobs${query ? '?' + query : ''}`)
   },
    diagnosticsExport: () =>
      request<ExportResponse>('/api/diagnostics/export', {
        method: 'POST',
      }),

    // Full config API (Phase 11)
    fullConfig: () => request<FullAppConfig>('/api/config'),
    validateFullConfig: (cfg: FullAppConfig) =>
      request<ExtendedConfigValidationResult>('/api/config/validate', {
        method: 'POST',
        body: JSON.stringify(cfg),
      }),
    applyFullConfig: (cfg: FullAppConfig) =>
      request<ExtendedConfigValidationResult>('/api/config/apply', {
        method: 'POST',
        body: JSON.stringify(cfg),
      }),
    previewFullConfig: (cfg?: FullAppConfig) =>
      request<string>(cfg ? '/api/config/preview' : '/api/config/preview', {
        method: cfg ? 'POST' : 'GET',
        body: cfg ? JSON.stringify(cfg) : undefined,
      }),
    createConfigSnapshot: (label?: string) =>
      request<ConfigSnapshot>('/api/config/backup', {
        method: 'POST',
        body: JSON.stringify({ label: label ?? '' }),
      }),
    listConfigSnapshots: () => request<ConfigSnapshotList>('/api/config/backups'),
    restoreConfigSnapshot: (id: string) =>
      request<{ restoredSnapshotId: string; preventiveSnapshotId: string }>(
        `/api/config/backups/${id}/restore`,
        { method: 'POST' }
      ),
    importSettingsJS: (content: string) =>
      request<{ config: FullAppConfig; warnings: string[] }>('/api/config/import', {
        method: 'POST',
        body: JSON.stringify({ content }),
      }),

    // Asset management
    uploadAsset: (category: string, file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return requestMultipart<AssetInfo>(`/api/assets/${category}/upload`, formData)
    },
    listAssets: (category: string) => request<AssetList>(`/api/assets/${category}`),
    deleteAsset: (category: string, id: string) =>
      request<{ deleted: boolean }>(`/api/assets/${category}/${id}`, {
        method: 'DELETE',
      }),
}
