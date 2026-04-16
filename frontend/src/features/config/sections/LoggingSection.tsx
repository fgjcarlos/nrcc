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
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Observability</p>
        <h3 className="config-section-title">Logging</h3>
        <p className="config-section-copy">
          Set console verbosity and decide whether performance metrics and audit events are emitted alongside runtime logs.
        </p>
      </div>

       <div className="config-section-card space-y-3">
          <label className="label">
              <span className="label-text font-medium">Console Level</span>
          </label>
         <select
           className={`select select-bordered bg-base-100${errors['logging.console.level'] ? ' select-error' : ''}`}
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
            <span className="form-field-error-msg">
             <svg className="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
               <path fillRule="evenodd" d="M18.101 12.93a1 1 0 00-1.414-1.414L11 14.586l-2.687-2.687a1 1 0 00-1.414 1.414l4.1 4.1a1 1 0 001.414 0l8.101-8.101z" clipRule="evenodd" />
               <path fillRule="evenodd" d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 14a6 6 0 110-12 6 6 0 010 12z" clipRule="evenodd" />
             </svg>
             <span>{errors['logging.console.level']}</span>
            </span>
          )}
        </div>

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.console.metrics}
            onChange={(e) => updateConsole('metrics', e.target.checked)}
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Log Metrics</span>
            <span className="config-toggle-hint">Include performance telemetry and timing data in console output.</span>
          </span>
      </label>

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.console.audit}
            onChange={(e) => updateConsole('audit', e.target.checked)}
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Log Audit Events</span>
            <span className="config-toggle-hint">Record administrative API activity for traceability and support reviews.</span>
          </span>
      </label>
    </article>
  )
}
