import { ServerConfig } from '../../../types/config'
import { FormField } from '../../../components/forms'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

export function ServerSection({ value, onChange, errors }: SectionProps<ServerConfig>) {
  const updateField = <K extends keyof ServerConfig>(key: K, val: ServerConfig[K]) => {
    onChange({ ...value, [key]: val })
  }

  return (
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">Server</h3>

      <FormField
        id="server-ui-port"
        label="UI Port"
        type="number"
        value={value.uiPort}
        onChange={(v) => updateField('uiPort', parseInt(v) || 1880)}
        error={errors['server.uiPort']}
        min={1}
        max={65535}
      />

      <FormField
        id="server-ui-host"
        label="UI Host"
        type="text"
        value={value.uiHost}
        onChange={(v) => updateField('uiHost', v)}
        placeholder="0.0.0.0"
        error={errors['server.uiHost']}
      />

      <FormField
        id="server-http-admin-root"
        label="HTTP Admin Root"
        type="text"
        value={value.httpAdminRoot}
        onChange={(v) => updateField('httpAdminRoot', v)}
        placeholder="/"
        error={errors['server.httpAdminRoot']}
      />

      <FormField
        id="server-http-node-root"
        label="HTTP Node Root"
        type="text"
        value={value.httpNodeRoot}
        onChange={(v) => updateField('httpNodeRoot', v)}
        placeholder="/"
        hint="Set to 'false' to disable"
        error={errors['server.httpNodeRoot']}
      />

      <FormField
        id="server-http-static"
        label="HTTP Static"
        type="text"
        value={value.httpStatic}
        onChange={(v) => updateField('httpStatic', v)}
        placeholder="/path/to/static"
        error={errors['server.httpStatic']}
      />

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.disableEditor}
            onChange={(e) => updateField('disableEditor', e.target.checked)}
          />
          <span className="label-text font-medium">Disable Editor</span>
        </label>
        {value.disableEditor && (
          <p className="text-warning text-sm mt-2">⚠️ Disables the Node-RED editor UI</p>
        )}
      </div>
    </article>
  )
}
