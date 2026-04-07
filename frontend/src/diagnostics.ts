// Types for diagnostics feature

export type DoctorCheckStatus = 'pass' | 'warn' | 'fail' | 'unknown'

export interface DoctorCheck {
  name: string
  status: DoctorCheckStatus
  message: string
  details?: Record<string, unknown>
}

export interface DoctorReport {
  generated_at: string
  overall_status: DoctorCheckStatus
  checks: DoctorCheck[]
}

export type LogLevel = 'debug' | 'info' | 'warn' | 'error'

export interface LogEntry {
  id?: string
  timestamp: string
  level: LogLevel
  source: string
  event?: string
  message: string
  metadata?: Record<string, unknown>
}

export interface LogsResponse {
  logs: LogEntry[]
  total: number
}

export type JobStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface JobRecord {
  id: string
  type: string
  status: JobStatus
  started_at: string
  finished_at?: string
  triggered_by?: string
  summary?: string
  error?: string
}

export interface JobsResponse {
  jobs: JobRecord[]
  total: number
}

export interface ExportResponse {
  path: string
  size: number
}
