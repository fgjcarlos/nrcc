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
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">Flows</h3>

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

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.flowFilePretty}
            onChange={(e) => updateField('flowFilePretty', e.target.checked)}
          />
          <span className="label-text font-medium">Pretty-print flows.json</span>
        </label>
      </div>

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
    </article>
  )
}
