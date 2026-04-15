import { useEffect, useState, useRef } from 'react'
import { api } from '../../api'
import { FullAppConfig } from '../../types/config'

type LivePreviewPanelProps = {
  config: FullAppConfig
  isOpen: boolean
  onToggle: () => void
}

export function LivePreviewPanel({ config, isOpen, onToggle }: LivePreviewPanelProps) {
  const [preview, setPreview] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Debounced preview fetch
  useEffect(() => {
    if (!isOpen) return

    // Clear existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current)
    }

    setLoading(true)
    setError('')

    // Set new debounce timer
    debounceTimerRef.current = setTimeout(async () => {
      try {
        const result = await api.previewFullConfig(config)
        setPreview(result)
        setError('')
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch preview')
        setPreview('')
      } finally {
        setLoading(false)
      }
    }, 800)

    // Cleanup timer on unmount
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current)
      }
    }
  }, [config, isOpen])

  if (!isOpen) {
    return null
  }

  return (
    <div className="card bg-base-200 fixed inset-4 z-40 overflow-auto md:inset-auto md:w-1/2 md:right-4 md:bottom-4">
      <div className="card-body">
        <div className="flex items-center justify-between mb-4">
          <h3 className="card-title text-lg">Preview settings.js</h3>
          <button className="btn btn-ghost btn-sm btn-circle" onClick={onToggle} aria-label="Close preview">
            ✕
          </button>
        </div>

        {loading && <p className="text-sm text-base-content/60">Loading preview...</p>}

        {error && (
          <p className="alert alert-error">
            <strong>Error:</strong> {error}
          </p>
        )}

        {!loading && !error && preview && (
          <pre className="bg-base-300 p-4 rounded overflow-x-auto text-xs">
            <code>{preview}</code>
          </pre>
        )}
      </div>
    </div>
  )
}
