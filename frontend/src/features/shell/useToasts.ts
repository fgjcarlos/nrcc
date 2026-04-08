import { useState } from 'react'
import type { Toast } from '../../common/types'

export function useToasts() {
  const [toasts, setToasts] = useState<Toast[]>([])

  function pushToast(toast: Omit<Toast, 'id'>) {
    const id = Date.now() + Math.floor(Math.random() * 1000)
    setToasts((current) => [...current, { ...toast, id }])
  }

  function dismissToast(id: number) {
    setToasts((current) => current.filter((toast) => toast.id !== id))
  }

  return { toasts, pushToast, dismissToast }
}
