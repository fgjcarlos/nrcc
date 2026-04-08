import { LoggingConfig } from '../../../types/config'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

const LOG_LEVELS = ['fatal', 'error', 'warn', 'info', 'debug', 'trace'] as const

export function LoggingSection({ value, onChange, errors }: SectionProps<LoggingConfig>) {
  const updateConsole = (key: keyof typeof value.console, val: any) => {
    onChange({
      ...value,
      console: { ...value.console, [key]: val },
    })
  }

  return (
    <article className="settings-section">
      <h3>Logging</h3>

      <label className="form-field">
        <span>Console Level</span>
        <select
          value={value.console.level}
          onChange={(e) => updateConsole('level', e.target.value)}
        >
          {LOG_LEVELS.map((level) => (
            <option key={level} value={level}>
              {level.charAt(0).toUpperCase() + level.slice(1)}
            </option>
          ))}
        </select>
        {errors['logging.console.level'] && (
          <p className="field-error">{errors['logging.console.level']}</p>
        )}
      </label>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.console.metrics}
          onChange={(e) => updateConsole('metrics', e.target.checked)}
        />
        <span>Log Metrics (performance data)</span>
      </label>

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.console.audit}
          onChange={(e) => updateConsole('audit', e.target.checked)}
        />
        <span>Log Audit Events (API calls)</span>
      </label>
    </article>
  )
}
