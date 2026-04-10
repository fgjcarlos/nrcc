import { useEffect } from 'react'
import type { Toast } from '../../common/types'

export function ToastViewport({
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

  const getToneClass = (tone: string) => {
    switch (tone) {
      case 'error':
        return 'alert-error'
      case 'success':
        return 'alert-success'
      case 'info':
      default:
        return 'alert-info'
    }
  }

  return (
    <div className="toast toast-top toast-right" aria-live="polite" aria-atomic="true">
      {toasts.map((toast) => (
        <article key={toast.id} className={`alert ${getToneClass(toast.tone)} shadow-lg`}>
          <div className="flex-1">
            <strong className="text-sm">{toast.title}</strong>
            {toast.detail ? <p className="text-xs opacity-90 mt-1">{toast.detail}</p> : null}
          </div>
          <button
            type="button"
            className="btn btn-ghost btn-xs"
            onClick={() => onDismiss(toast.id)}
            aria-label="Close notification"
          >
            ✕
          </button>
        </article>
      ))}
    </div>
  )
}
