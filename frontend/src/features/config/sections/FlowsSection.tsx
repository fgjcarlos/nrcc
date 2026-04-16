import { FlowsConfig } from '../../../types/config'
import { FormField } from '../../../components/forms'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

export function FlowsSection({ value, onChange, errors }: SectionProps<FlowsConfig>) {
  const updateField = <K extends keyof FlowsConfig>(key: K, val: FlowsConfig[K]) => {
    onChange({ ...value, [key]: val })
  }

  return (
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Runtime files</p>
        <h3 className="config-section-title">Flows</h3>
        <p className="config-section-copy">
          Define where flow state and custom nodes are stored, and whether written files stay human-readable.
        </p>
      </div>

      <div className="config-field-grid config-field-grid--two">
        <FormField
          id="flows-flow-file"
          label="Flow File"
          type="text"
          value={value.flowFile}
          onChange={(v) => updateField('flowFile', v)}
          placeholder="flows.json"
          hint="Filename only, no path separators"
          error={errors['flows.flowFile']}
        />

        <FormField
          id="flows-user-dir"
          label="User Directory"
          type="text"
          value={value.userDir}
          onChange={(v) => updateField('userDir', v)}
          placeholder="/absolute/path/to/user/dir"
          hint="Absolute path"
          error={errors['flows.userDir']}
        />

        <FormField
          id="flows-nodes-dir"
          label="Nodes Directory"
          type="text"
          value={value.nodesDir}
          onChange={(v) => updateField('nodesDir', v)}
          placeholder="/absolute/path/to/nodes"
          hint="Absolute path"
          error={errors['flows.nodesDir']}
        />
      </div>

      <label className="config-toggle-row cursor-pointer">
        <input
          type="checkbox"
          className="checkbox"
          checked={value.flowFilePretty}
          onChange={(e) => updateField('flowFilePretty', e.target.checked)}
        />
        <span className="config-toggle-copy">
          <span className="config-toggle-title">Pretty-print flows.json</span>
          <span className="config-toggle-hint">Keeps generated flow files easier to inspect and diff during manual troubleshooting.</span>
        </span>
      </label>
    </article>
  )
}
