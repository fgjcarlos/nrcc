export type AuthMode = 'login' | 'register'
export type PageKey = 'overview' | 'logs' | 'config' | 'environment' | 'backups' | 'libraries' | 'updates' | 'diagnostics' | 'users'
export type ToastTone = 'success' | 'error' | 'info'

export type Toast = {
  id: number
  title: string
  detail?: string
  tone: ToastTone
}

export type GlobalStatus = { title: string; detail: string; tone: 'ok' | 'warn' | 'neutral' }
