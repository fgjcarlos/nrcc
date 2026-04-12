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

      <div className="form-control">
        <label className="label">
          <span className="label-text font-medium">Function Timeout</span>
        </label>
        <div className="flex gap-2 items-center">
          <input
            type="number"
            className={`input input-bordered bg-base-100 flex-1${errors['runtime.functionTimeout'] ? ' input-error' : ''}`}
            value={value.functionTimeout}
            onChange={(e) => updateField('functionTimeout', parseInt(e.target.value) || 0)}
            min={0}
            max={3600}
            aria-describedby={errors['runtime.functionTimeout'] ? 'runtime-functionTimeout-error' : undefined}
          />
          <span className="text-base-content/60 text-sm min-w-max">seconds</span>
        </div>
        <p className="text-base-content/60 text-sm mt-1">0 = no timeout</p>
        {errors['runtime.functionTimeout'] && (
          <span id="runtime-functionTimeout-error" className="form-field-error-msg">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
            </svg>
            {errors['runtime.functionTimeout']}
          </span>
        )}
      </div>

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
