import { NodeReconnectConfig } from '../../../types/config'
import { FormField } from '../../../components/forms'

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
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Connectivity</p>
        <h3 className="config-section-title">Node Reconnect</h3>
        <p className="config-section-copy">
          Tune reconnect pacing for common network-facing nodes so transient outages recover cleanly.
        </p>
      </div>

      <div className="config-field-grid config-field-grid--two">
        <FormField
          id="nodeReconnect-mqttReconnectTime"
          label="MQTT Reconnect Time"
          type="number"
          value={value.mqttReconnectTime}
          onChange={(v) => updateField('mqttReconnectTime', parseInt(v) || 5000)}
          min={100}
          max={300000}
          unit="ms"
          error={errors['nodeReconnect.mqttReconnectTime']}
        />

        <FormField
          id="nodeReconnect-serialReconnectTime"
          label="Serial Reconnect Time"
          type="number"
          value={value.serialReconnectTime}
          onChange={(v) => updateField('serialReconnectTime', parseInt(v) || 5000)}
          min={100}
          max={300000}
          unit="ms"
          error={errors['nodeReconnect.serialReconnectTime']}
        />

        <FormField
          id="nodeReconnect-socketReconnectTime"
          label="Socket Reconnect Time"
          type="number"
          value={value.socketReconnectTime}
          onChange={(v) => updateField('socketReconnectTime', parseInt(v) || 10000)}
          min={100}
          max={300000}
          unit="ms"
          error={errors['nodeReconnect.socketReconnectTime']}
        />

        <FormField
          id="nodeReconnect-socketTimeout"
          label="Socket Timeout"
          type="number"
          value={value.socketTimeout}
          onChange={(v) => updateField('socketTimeout', parseInt(v) || 120000)}
          min={1000}
          max={600000}
          unit="ms"
          error={errors['nodeReconnect.socketTimeout']}
        />
      </div>

      <div className="config-chip-row">
        {presets.map((p) => (
          <button
            key={p.value}
            type="button"
            onClick={() => updateField('mqttReconnectTime', p.value)}
            className={`config-chip-btn ${value.mqttReconnectTime === p.value ? 'config-chip-btn-active' : ''}`}
          >
            {p.label}
          </button>
        ))}
      </div>
    </article>
  )
}
