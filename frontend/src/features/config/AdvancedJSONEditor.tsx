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
    <div className="advanced-json-editor modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h2>Raw JSON Editor</h2>
          <button className="close-button" onClick={onClose} aria-label="Close editor">
            ✕
          </button>
        </div>

        <div className="modal-body">
          <label className="form-field">
            <span>Configuration JSON</span>
            <textarea
              value={jsonText}
              onChange={(e) => {
                setJsonText(e.target.value)
                setJsonError('')
              }}
              className="json-textarea"
              rows={20}
            />
          </label>

          {jsonError && (
            <p className="field-error">
              <strong>Error:</strong> {jsonError}
            </p>
          )}
        </div>

        <div className="modal-footer">
          <button className="secondary-button" onClick={handleBackToForm}>
            Apply & Back to Form
          </button>
          <button className="primary-button" onClick={handleApply}>
            Apply JSON
          </button>
          <button className="secondary-button" onClick={onClose}>
            Cancel
          </button>
        </div>
      </div>
    </div>
  )
}
