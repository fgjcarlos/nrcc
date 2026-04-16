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
    <div className="modal-overlay">
      <div className="surface-card w-full max-w-2xl max-h-[32rem] overflow-auto border border-base-300/60 p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-base-content">Import from settings.js</h2>
          <button className="action-btn-ghost px-3 py-2" onClick={onClose} aria-label="Close import">
            ✕
          </button>
        </div>

        <div className="space-y-4">
          <div className="surface-panel section-tabbar border border-base-300/60">
            <button
              className={`section-tab ${tab === 'paste' ? 'section-tab-active' : ''}`}
              onClick={() => {
                setTab('paste')
                setError('')
              }}
            >
              Paste
            </button>
            <button
              className={`section-tab ${tab === 'upload' ? 'section-tab-active' : ''}`}
              onClick={() => {
                setTab('upload')
                setError('')
              }}
            >
              Upload File
            </button>
          </div>

          {tab === 'paste' && (
            <label className="form-field">
              <span className="label-text">Paste settings.js content</span>
              <textarea
                value={content}
                onChange={(e) => {
                  setContent(e.target.value)
                  setParsedConfig(null)
                  setWarnings([])
                }}
                className="textarea textarea-bordered mt-2 min-h-64 font-mono text-sm"
                rows={15}
                placeholder="module.exports = { ... }"
                disabled={loading}
              />
            </label>
          )}

          {tab === 'upload' && (
            <label className="form-field">
              <span className="label-text">Upload settings.js file</span>
              <input
                className="file-input file-input-bordered mt-2"
                type="file"
                onChange={handleFileUpload}
                accept=".js"
                disabled={loading}
              />
              {content && (
                <p className="form-field-hint">File content loaded. Click Import to parse.</p>
              )}
            </label>
          )}

          {error && (
            <p className="alert alert-error text-sm">
              <strong>Error:</strong> {error}
            </p>
          )}

          {/* Parsed result */}
          {parsedConfig && (
            <div className="surface-panel border border-base-300/60 p-4">
              <p className="font-medium text-success">Settings imported successfully</p>
              {warnings.length > 0 && (
                <div className="mt-3 text-sm text-base-content/75">
                  <strong>Warnings:</strong>
                  <ul className="mt-2 list-disc pl-5">
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
              className="action-btn-primary w-full"
              onClick={handleImport}
              disabled={loading || !content.trim()}
            >
              {loading ? 'Importing...' : 'Import'}
            </button>
          )}
        </div>

        {parsedConfig && (
          <div className="flex gap-3 justify-end mt-6">
            <button className="action-btn-ghost" onClick={onClose}>
              Cancel
            </button>
            <button
              className="action-btn-primary"
              onClick={handleApply}
            >
              Apply to Forms
            </button>
          </div>
        )}

        {!parsedConfig && (
          <div className="flex gap-3 justify-end mt-6">
            <button className="action-btn-ghost" onClick={onClose}>
              Close
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
