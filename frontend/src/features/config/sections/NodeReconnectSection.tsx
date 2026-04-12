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
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">Node Reconnect</h3>

      <div className="form-control">
        <label className="label">
          <span className="label-text font-medium">MQTT Reconnect Time</span>
        </label>
        <div className="flex gap-2 items-center">
          <input
            type="number"
            className={`input input-bordered bg-base-100 flex-1${errors['nodeReconnect.mqttReconnectTime'] ? ' input-error' : ''}`}
            value={value.mqttReconnectTime}
            onChange={(e) => updateField('mqttReconnectTime', parseInt(e.target.value) || 5000)}
            min={100}
            max={300000}
            aria-describedby={errors['nodeReconnect.mqttReconnectTime'] ? 'nodeReconnect-mqttReconnectTime-error' : undefined}
          />
          <span className="text-base-content/60 text-sm min-w-max">ms</span>
        </div>
        <div className="flex flex-wrap gap-2 mt-3">
          {presets.map((p) => (
            <button
              key={p.value}
              type="button"
              onClick={() => updateField('mqttReconnectTime', p.value)}
              className={`btn btn-sm ${value.mqttReconnectTime === p.value ? 'btn-primary' : 'btn-ghost'}`}
            >
              {p.label}
            </button>
          ))}
        </div>
        {errors['nodeReconnect.mqttReconnectTime'] && (
          <span id="nodeReconnect-mqttReconnectTime-error" className="form-field-error-msg">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
            </svg>
            {errors['nodeReconnect.mqttReconnectTime']}
          </span>
        )}
      </div>

      <div className="form-control">
        <label className="label">
          <span className="label-text font-medium">Serial Reconnect Time</span>
        </label>
        <div className="flex gap-2 items-center">
          <input
            type="number"
            className={`input input-bordered bg-base-100 flex-1${errors['nodeReconnect.serialReconnectTime'] ? ' input-error' : ''}`}
            value={value.serialReconnectTime}
            onChange={(e) => updateField('serialReconnectTime', parseInt(e.target.value) || 5000)}
            min={100}
            max={300000}
            aria-describedby={errors['nodeReconnect.serialReconnectTime'] ? 'nodeReconnect-serialReconnectTime-error' : undefined}
          />
          <span className="text-base-content/60 text-sm min-w-max">ms</span>
        </div>
        {errors['nodeReconnect.serialReconnectTime'] && (
          <span id="nodeReconnect-serialReconnectTime-error" className="form-field-error-msg">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
            </svg>
            {errors['nodeReconnect.serialReconnectTime']}
          </span>
        )}
      </div>

      <div className="form-control">
        <label className="label">
          <span className="label-text font-medium">Socket Reconnect Time</span>
        </label>
        <div className="flex gap-2 items-center">
          <input
            type="number"
            className={`input input-bordered bg-base-100 flex-1${errors['nodeReconnect.socketReconnectTime'] ? ' input-error' : ''}`}
            value={value.socketReconnectTime}
            onChange={(e) => updateField('socketReconnectTime', parseInt(e.target.value) || 10000)}
            min={100}
            max={300000}
            aria-describedby={errors['nodeReconnect.socketReconnectTime'] ? 'nodeReconnect-socketReconnectTime-error' : undefined}
          />
          <span className="text-base-content/60 text-sm min-w-max">ms</span>
        </div>
        {errors['nodeReconnect.socketReconnectTime'] && (
          <span id="nodeReconnect-socketReconnectTime-error" className="form-field-error-msg">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
            </svg>
            {errors['nodeReconnect.socketReconnectTime']}
          </span>
        )}
      </div>

      <div className="form-control">
        <label className="label">
          <span className="label-text font-medium">Socket Timeout</span>
        </label>
        <div className="flex gap-2 items-center">
          <input
            type="number"
            className={`input input-bordered bg-base-100 flex-1${errors['nodeReconnect.socketTimeout'] ? ' input-error' : ''}`}
            value={value.socketTimeout}
            onChange={(e) => updateField('socketTimeout', parseInt(e.target.value) || 120000)}
            min={1000}
            max={600000}
            aria-describedby={errors['nodeReconnect.socketTimeout'] ? 'nodeReconnect-socketTimeout-error' : undefined}
          />
          <span className="text-base-content/60 text-sm min-w-max">ms</span>
        </div>
        {errors['nodeReconnect.socketTimeout'] && (
          <span id="nodeReconnect-socketTimeout-error" className="form-field-error-msg">
            <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
            </svg>
            {errors['nodeReconnect.socketTimeout']}
          </span>
        )}
      </div>
    </article>
  )
}
