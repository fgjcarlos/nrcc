import { NodeReconnectConfig } from '../../../types/config'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

export function NodeReconnectSection({ value, onChange, errors }: SectionProps<NodeReconnectConfig>) {
  const updateField = <K extends keyof NodeReconnectConfig>(
    key: K,
    val: NodeReconnectConfig[K]
  ) => {
    onChange({ ...value, [key]: val })
  }

  const presets = [
    { label: '1s', value: 1000 },
    { label: '5s', value: 5000 },
    { label: '10s', value: 10000 },
    { label: '30s', value: 30000 },
  ]

  return (
    <article className="settings-section">
      <h3>Node Reconnect</h3>

      <label className="form-field">
        <span>MQTT Reconnect Time</span>
        <div className="input-group">
          <input
            type="number"
            value={value.mqttReconnectTime}
            onChange={(e) => updateField('mqttReconnectTime', parseInt(e.target.value) || 5000)}
            min={100}
            max={300000}
          />
          <span className="unit">ms</span>
        </div>
        <div className="preset-buttons">
          {presets.map((p) => (
            <button
              key={p.value}
              type="button"
              onClick={() => updateField('mqttReconnectTime', p.value)}
              className={`preset-button ${value.mqttReconnectTime === p.value ? 'active' : ''}`}
            >
              {p.label}
            </button>
          ))}
        </div>
        {errors['nodeReconnect.mqttReconnectTime'] && (
          <p className="field-error">{errors['nodeReconnect.mqttReconnectTime']}</p>
        )}
      </label>

      <label className="form-field">
        <span>Serial Reconnect Time</span>
        <div className="input-group">
          <input
            type="number"
            value={value.serialReconnectTime}
            onChange={(e) => updateField('serialReconnectTime', parseInt(e.target.value) || 5000)}
            min={100}
            max={300000}
          />
          <span className="unit">ms</span>
        </div>
        {errors['nodeReconnect.serialReconnectTime'] && (
          <p className="field-error">{errors['nodeReconnect.serialReconnectTime']}</p>
        )}
      </label>

      <label className="form-field">
        <span>Socket Reconnect Time</span>
        <div className="input-group">
          <input
            type="number"
            value={value.socketReconnectTime}
            onChange={(e) => updateField('socketReconnectTime', parseInt(e.target.value) || 10000)}
            min={100}
            max={300000}
          />
          <span className="unit">ms</span>
        </div>
        {errors['nodeReconnect.socketReconnectTime'] && (
          <p className="field-error">{errors['nodeReconnect.socketReconnectTime']}</p>
        )}
      </label>

      <label className="form-field">
        <span>Socket Timeout</span>
        <div className="input-group">
          <input
            type="number"
            value={value.socketTimeout}
            onChange={(e) => updateField('socketTimeout', parseInt(e.target.value) || 120000)}
            min={1000}
            max={600000}
          />
          <span className="unit">ms</span>
        </div>
        {errors['nodeReconnect.socketTimeout'] && (
          <p className="field-error">{errors['nodeReconnect.socketTimeout']}</p>
        )}
      </label>
    </article>
  )
}
