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
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">Runtime</h3>

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.functionExternalModules}
            onChange={(e) => updateField('functionExternalModules', e.target.checked)}
          />
          <span className="label-text font-medium">Allow External Modules in Function Nodes</span>
        </label>
      </div>

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

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.diagnosticsEnabled}
            onChange={(e) => updateField('diagnosticsEnabled', e.target.checked)}
          />
          <span className="label-text font-medium">Enable Diagnostics</span>
        </label>
      </div>

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.safeMode}
            onChange={(e) => updateField('safeMode', e.target.checked)}
          />
          <span className="label-text font-medium">Safe Mode</span>
        </label>
        {value.safeMode && (
          <p className="text-warning text-sm mt-2">⚠️ Safe mode starts Node-RED without running flows</p>
        )}
      </div>

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
    </article>
  )
}
