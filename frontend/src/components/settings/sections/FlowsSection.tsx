import { FlowsConfig } from '../../../types/config'

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
    <article className="settings-section">
      <h3>Flows</h3>

      <label className="form-field">
        <span>Flow File</span>
        <input
          type="text"
          value={value.flowFile}
          onChange={(e) => updateField('flowFile', e.target.value)}
          placeholder="flows.json"
        />
        <p className="field-hint">Filename only, no path separators</p>
        {errors['flows.flowFile'] && <p className="field-error">{errors['flows.flowFile']}</p>}
      </label>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.flowFilePretty}
          onChange={(e) => updateField('flowFilePretty', e.target.checked)}
        />
        <span>Pretty-print flows.json</span>
      </label>

      <label className="form-field">
        <span>User Directory</span>
        <input
          type="text"
          value={value.userDir}
          onChange={(e) => updateField('userDir', e.target.value)}
          placeholder="/absolute/path/to/user/dir"
        />
        <p className="field-hint">Absolute path</p>
        {errors['flows.userDir'] && <p className="field-error">{errors['flows.userDir']}</p>}
      </label>

      <label className="form-field">
        <span>Nodes Directory</span>
        <input
          type="text"
          value={value.nodesDir}
          onChange={(e) => updateField('nodesDir', e.target.value)}
          placeholder="/absolute/path/to/nodes"
        />
        <p className="field-hint">Absolute path</p>
        {errors['flows.nodesDir'] && <p className="field-error">{errors['flows.nodesDir']}</p>}
      </label>
    </article>
  )
}
