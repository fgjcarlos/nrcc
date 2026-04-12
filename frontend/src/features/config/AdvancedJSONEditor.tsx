import { useState } from 'react'
import { FullAppConfig } from '../../types/config'

type AdvancedJSONEditorProps = {
  config: FullAppConfig
  onApply: (cfg: FullAppConfig) => void
  onClose: () => void
}

export function AdvancedJSONEditor({ config, onApply, onClose }: AdvancedJSONEditorProps) {
  const [jsonText, setJsonText] = useState(() => JSON.stringify(config, null, 2))
  const [jsonError, setJsonError] = useState<string>('')

  const handleApply = () => {
    try {
      const parsed = JSON.parse(jsonText)
      // Basic validation: check if it has the required top-level keys
      const requiredKeys = [
        'server',
        'security',
        'editorTheme',
        'flows',
        'contextStorage',
        'logging',
        'runtime',
        'https',
        'nodeReconnect',
        'palette',
      ]
      const hasAllKeys = requiredKeys.every((key) => key in parsed)
      if (!hasAllKeys) {
        setJsonError(
          'Invalid configuration: missing required sections (' +
            requiredKeys.filter((k) => !(k in parsed)).join(', ') +
            ')'
        )
        return
      }
      setJsonError('')
      onApply(parsed as FullAppConfig)
      onClose()
    } catch (err) {
      setJsonError(err instanceof Error ? err.message : 'Invalid JSON')
    }
  }

  const handleBackToForm = () => {
    try {
      const parsed = JSON.parse(jsonText)
      onApply(parsed as FullAppConfig)
      onClose()
    } catch (err) {
      setJsonError(err instanceof Error ? err.message : 'Invalid JSON')
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
      <div className="card bg-base-200 shadow-lg w-full max-w-2xl max-h-96 overflow-auto">
        <div className="card-body">
          <div className="flex items-center justify-between mb-4">
            <h2 className="card-title text-xl">Raw JSON Editor</h2>
            <button className="btn btn-ghost btn-sm btn-circle" onClick={onClose} aria-label="Close editor">
              ✕
            </button>
          </div>

          <div className="form-control">
            <label className="label">
              <span className="label-text font-medium">Configuration JSON</span>
            </label>
            <textarea
              value={jsonText}
              onChange={(e) => {
                setJsonText(e.target.value)
                setJsonError('')
              }}
              className="textarea textarea-bordered bg-base-100 font-mono text-sm"
              rows={15}
            />
          </div>

          {jsonError && (
            <p className="alert alert-error text-sm">
              <strong>Error:</strong> {jsonError}
            </p>
          )}

          <div className="flex gap-3 justify-end mt-6">
            <button className="btn btn-ghost" onClick={handleBackToForm}>
              Apply & Back to Form
            </button>
            <button className="btn btn-primary" onClick={handleApply}>
              Apply JSON
            </button>
            <button className="btn btn-ghost" onClick={onClose}>
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
