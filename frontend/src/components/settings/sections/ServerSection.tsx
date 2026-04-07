import { ServerConfig } from '../../../types/config'

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
    <article className="settings-section">
      <h3>Server</h3>

      <label className="form-field">
        <span>UI Port</span>
        <input
          type="number"
          value={value.uiPort}
          onChange={(e) => updateField('uiPort', parseInt(e.target.value) || 1880)}
          min={1}
          max={65535}
        />
        {errors['server.uiPort'] && <p className="field-error">{errors['server.uiPort']}</p>}
      </label>

      <label className="form-field">
        <span>UI Host</span>
        <input
          type="text"
          value={value.uiHost}
          onChange={(e) => updateField('uiHost', e.target.value)}
          placeholder="0.0.0.0"
        />
        {errors['server.uiHost'] && <p className="field-error">{errors['server.uiHost']}</p>}
      </label>

      <label className="form-field">
        <span>HTTP Admin Root</span>
        <input
          type="text"
          value={value.httpAdminRoot}
          onChange={(e) => updateField('httpAdminRoot', e.target.value)}
          placeholder="/"
        />
        {errors['server.httpAdminRoot'] && <p className="field-error">{errors['server.httpAdminRoot']}</p>}
      </label>

      <label className="form-field">
        <span>HTTP Node Root</span>
        <input
          type="text"
          value={value.httpNodeRoot}
          onChange={(e) => updateField('httpNodeRoot', e.target.value)}
          placeholder="/"
        />
        <p className="field-hint">Set to 'false' to disable</p>
        {errors['server.httpNodeRoot'] && <p className="field-error">{errors['server.httpNodeRoot']}</p>}
      </label>

      <label className="form-field">
        <span>HTTP Static</span>
        <input
          type="text"
          value={value.httpStatic}
          onChange={(e) => updateField('httpStatic', e.target.value)}
          placeholder="/path/to/static"
        />
        {errors['server.httpStatic'] && <p className="field-error">{errors['server.httpStatic']}</p>}
      </label>

      <label className="form-field form-toggle">
        <input
          type="checkbox"
          checked={value.disableEditor}
          onChange={(e) => updateField('disableEditor', e.target.checked)}
        />
        <span>Disable Editor</span>
        {value.disableEditor && (
          <p className="field-warning">⚠️ Disables the Node-RED editor UI</p>
        )}
      </label>
    </article>
  )
}
