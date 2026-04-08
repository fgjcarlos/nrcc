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
