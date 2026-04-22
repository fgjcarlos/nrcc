export type AuthMode = 'login' | 'register'
export type PageKey = 'overview' | 'logs' | 'flows' | 'config' | 'environment' | 'backups' | 'libraries' | 'updates' | 'diagnostics'
export type ToastTone = 'success' | 'error' | 'info'

export type Toast = {
  id: number
  title: string
  detail?: string
  tone: ToastTone
}

export type GlobalStatus = { title: string; detail: string; tone: 'ok' | 'warn' | 'neutral' }

export type ExportRequest = {
  ids: string[]
}

export type ImportResponse = {
  importedCount: number
  message: string
  restartAdvisory: boolean
}
