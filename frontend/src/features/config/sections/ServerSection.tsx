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
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Runtime</p>
        <h3 className="config-section-title">Server</h3>
        <p className="config-section-copy">
          Control the Node-RED listener, editor roots, and static asset mounting points.
        </p>
      </div>

      <div className="config-field-grid config-field-grid--two">
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

        <div className="md:col-span-2">
          <FormField
            id="server-http-static"
            label="HTTP Static"
            type="text"
            value={value.httpStatic}
            onChange={(v) => updateField('httpStatic', v)}
            placeholder="/path/to/static"
            error={errors['server.httpStatic']}
          />
        </div>
      </div>

      <label className="config-toggle-row cursor-pointer">
        <input
          type="checkbox"
          className="checkbox"
          checked={value.disableEditor}
          onChange={(e) => updateField('disableEditor', e.target.checked)}
        />
        <span className="config-toggle-copy">
          <span className="config-toggle-title">Disable Editor</span>
          <span className="config-toggle-hint">Turns off the Node-RED editor UI while keeping runtime endpoints available.</span>
        </span>
      </label>
    </article>
  )
}
