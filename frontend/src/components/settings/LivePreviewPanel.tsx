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
    <div className="live-preview-panel">
      <div className="panel-header">
        <h3>Preview settings.js</h3>
        <button className="close-button" onClick={onToggle} aria-label="Close preview">
          ✕
        </button>
      </div>

      {loading && <p className="muted">Loading preview...</p>}

      {error && (
        <p className="field-error">
          <strong>Error:</strong> {error}
        </p>
      )}

      {!loading && !error && preview && (
        <pre className="code-block">
          <code>{preview}</code>
        </pre>
      )}
    </div>
  )
}
