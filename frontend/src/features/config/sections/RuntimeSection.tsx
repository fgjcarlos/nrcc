import { RuntimeConfig } from '../../../types/config'
import { FormField } from '../../../components/forms'

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
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Execution</p>
        <h3 className="config-section-title">Runtime</h3>
        <p className="config-section-copy">
          Adjust execution limits, diagnostics, and fail-safe behavior for the active Node-RED runtime.
        </p>
      </div>

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.functionExternalModules}
            onChange={(e) => updateField('functionExternalModules', e.target.checked)}
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Allow External Modules in Function Nodes</span>
            <span className="config-toggle-hint">Lets function nodes require additional packages at runtime.</span>
          </span>
      </label>

      <div className="config-field-grid config-field-grid--two">
        <FormField
          id="runtime-function-timeout"
          label="Function Timeout"
          type="number"
          value={value.functionTimeout}
          onChange={(v) => updateField('functionTimeout', parseInt(v) || 0)}
          min={0}
          max={3600}
          hint="0 = no timeout"
          error={errors['runtime.functionTimeout']}
        />

        <FormField
          id="runtime-debug-max-length"
          label="Debug Max Length"
          type="number"
          value={value.debugMaxLength}
          onChange={(v) => updateField('debugMaxLength', parseInt(v) || 1000)}
          min={100}
          max={100000}
          error={errors['runtime.debugMaxLength']}
        />

        <FormField
          id="runtime-node-message-buffer-max-length"
          label="Node Message Buffer Max Length"
          type="number"
          value={value.nodeMessageBufferMaxLength}
          onChange={(v) => updateField('nodeMessageBufferMaxLength', parseInt(v) || 0)}
          min={0}
          max={10000}
          hint="0 = unlimited"
          error={errors['runtime.nodeMessageBufferMaxLength']}
        />
      </div>

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.diagnosticsEnabled}
            onChange={(e) => updateField('diagnosticsEnabled', e.target.checked)}
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Enable Diagnostics</span>
            <span className="config-toggle-hint">Expose additional runtime health and troubleshooting information to the control center.</span>
          </span>
      </label>

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.safeMode}
            onChange={(e) => updateField('safeMode', e.target.checked)}
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Safe Mode</span>
            <span className="config-toggle-hint">Starts Node-RED without running flows so broken deployments can be repaired.</span>
          </span>
      </label>
    </article>
  )
}
