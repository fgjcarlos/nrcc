import { RuntimeConfig } from '../../../types/config'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

export function RuntimeSection({ value, onChange, errors }: SectionProps<RuntimeConfig>) {
  const updateField = <K extends keyof RuntimeConfig>(key: K, val: RuntimeConfig[K]) => {
    onChange({ ...value, [key]: val })
  }

  return (
    <article className="settings-section">
      <h3>Runtime</h3>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.functionExternalModules}
          onChange={(e) => updateField('functionExternalModules', e.target.checked)}
        />
        <span>Allow External Modules in Function Nodes</span>
      </label>

      <label className="form-field">
        <span>Function Timeout</span>
        <div className="input-group">
          <input
            type="number"
            value={value.functionTimeout}
            onChange={(e) => updateField('functionTimeout', parseInt(e.target.value) || 0)}
            min={0}
            max={3600}
          />
          <span className="unit">seconds</span>
        </div>
        <p className="field-hint">0 = no timeout</p>
        {errors['runtime.functionTimeout'] && (
          <p className="field-error">{errors['runtime.functionTimeout']}</p>
        )}
      </label>

      <label className="form-field">
        <span>Debug Max Length</span>
        <input
          type="number"
          value={value.debugMaxLength}
          onChange={(e) => updateField('debugMaxLength', parseInt(e.target.value) || 1000)}
          min={100}
          max={100000}
        />
        {errors['runtime.debugMaxLength'] && (
          <p className="field-error">{errors['runtime.debugMaxLength']}</p>
        )}
      </label>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.diagnosticsEnabled}
          onChange={(e) => updateField('diagnosticsEnabled', e.target.checked)}
        />
        <span>Enable Diagnostics</span>
      </label>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.safeMode}
          onChange={(e) => updateField('safeMode', e.target.checked)}
        />
        <span>Safe Mode</span>
        {value.safeMode && (
          <p className="field-warning">⚠️ Safe mode starts Node-RED without running flows</p>
        )}
      </label>

      <label className="form-field">
        <span>Node Message Buffer Max Length</span>
        <input
          type="number"
          value={value.nodeMessageBufferMaxLength}
          onChange={(e) => updateField('nodeMessageBufferMaxLength', parseInt(e.target.value) || 0)}
          min={0}
          max={10000}
        />
        <p className="field-hint">0 = unlimited</p>
        {errors['runtime.nodeMessageBufferMaxLength'] && (
          <p className="field-error">{errors['runtime.nodeMessageBufferMaxLength']}</p>
        )}
      </label>
    </article>
  )
}
