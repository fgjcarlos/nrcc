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
    <div className="surface-card fixed inset-4 z-40 overflow-auto border border-base-300/60 p-5 md:inset-auto md:right-4 md:bottom-4 md:w-1/2">
      <div>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold">Preview settings.js</h3>
          <button className="action-btn-ghost px-3 py-2" onClick={onToggle} aria-label="Close preview">
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
          <pre className="code-block-bg overflow-x-auto text-xs">
            <code>{preview}</code>
          </pre>
        )}
      </div>
    </div>
  )
}
