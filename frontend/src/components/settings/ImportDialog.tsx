import { useState } from 'react'
import { api } from '../../api'
import { FullAppConfig } from '../../types/config'

type ImportDialogProps = {
  isOpen: boolean
  onClose: () => void
  onImported: (cfg: FullAppConfig) => void
}

type ImportTab = 'paste' | 'upload'

export function ImportDialog({ isOpen, onClose, onImported }: ImportDialogProps) {
  const [tab, setTab] = useState<ImportTab>('paste')
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')
  const [parsedConfig, setParsedConfig] = useState<FullAppConfig | null>(null)
  const [warnings, setWarnings] = useState<string[]>([])

  const handleImport = async () => {
    if (!content.trim()) {
      setError('Please paste or upload settings.js content')
      return
    }

    setLoading(true)
    setError('')
    setParsedConfig(null)
    setWarnings([])

    try {
      const result = await api.importSettingsJS(content)
      setParsedConfig(result.config)
      setWarnings(result.warnings)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to import settings.js')
    } finally {
      setLoading(false)
    }
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      const text = event.target?.result as string
      setContent(text)
    }
    reader.readAsText(file)
  }

  const handleApply = () => {
    if (!parsedConfig) return
    onImported(parsedConfig)
    onClose()
  }

  if (!isOpen) {
    return null
  }

  return (
    <div className="import-dialog modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h2>Import from settings.js</h2>
          <button className="close-button" onClick={onClose} aria-label="Close import">
            ✕
          </button>
        </div>

        <div className="modal-body">
          {/* Tab bar */}
          <div className="tab-bar">
            <button
              className={`tab ${tab === 'paste' ? 'active' : ''}`}
              onClick={() => {
                setTab('paste')
                setError('')
              }}
            >
              Paste
            </button>
            <button
              className={`tab ${tab === 'upload' ? 'active' : ''}`}
              onClick={() => {
                setTab('upload')
                setError('')
              }}
            >
              Upload File
            </button>
          </div>

          {/* Tab content */}
          {tab === 'paste' && (
            <label className="form-field">
              <span>Paste settings.js content</span>
              <textarea
                value={content}
                onChange={(e) => {
                  setContent(e.target.value)
                  setParsedConfig(null)
                  setWarnings([])
                }}
                className="json-textarea"
                rows={15}
                placeholder="module.exports = { ... }"
                disabled={loading}
              />
            </label>
          )}

          {tab === 'upload' && (
            <label className="form-field">
              <span>Upload settings.js file</span>
              <input
                type="file"
                onChange={handleFileUpload}
                accept=".js"
                disabled={loading}
              />
              {content && (
                <p className="field-hint">File content loaded. Click Import to parse.</p>
              )}
            </label>
          )}

          {error && (
            <p className="field-error">
              <strong>Error:</strong> {error}
            </p>
          )}

          {/* Parsed result */}
          {parsedConfig && (
            <div className="import-result">
              <p className="success-message">✓ Settings imported successfully</p>
              {warnings.length > 0 && (
                <div className="warnings">
                  <strong>Warnings:</strong>
                  <ul>
                    {warnings.map((w, i) => (
                      <li key={i}>{w}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}

          {/* Import button */}
          {!parsedConfig && (
            <button
              className="primary-button full-width"
              onClick={handleImport}
              disabled={loading || !content.trim()}
            >
              {loading ? 'Importing...' : 'Import'}
            </button>
          )}
        </div>

        {parsedConfig && (
          <div className="modal-footer">
            <button className="secondary-button" onClick={onClose}>
              Cancel
            </button>
            <button
              className="primary-button"
              onClick={handleApply}
            >
              Apply to Forms
            </button>
          </div>
        )}

        {!parsedConfig && (
          <div className="modal-footer">
            <button className="secondary-button" onClick={onClose}>
              Close
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
