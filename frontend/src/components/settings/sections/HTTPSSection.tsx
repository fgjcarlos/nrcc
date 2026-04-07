import { HTTPSConfig } from '../../../types/config'

type SectionProps<T> = {
  value: T
  onChange: (next: T) => void
  errors: Record<string, string>
}

export function HTTPSSection({ value, onChange, errors }: SectionProps<HTTPSConfig>) {
  const updateField = <K extends keyof HTTPSConfig>(key: K, val: HTTPSConfig[K]) => {
    onChange({ ...value, [key]: val })
  }

  return (
    <article className="settings-section">
      <h3>HTTPS</h3>

      {value.enabled && (
        <section className="inline-notice warn">
          <strong>HTTPS Enabled</strong>
          <p>
            After enabling HTTPS, you must restart Node-RED and update your browser URL to
            https://
          </p>
        </section>
      )}

      <label className="form-toggle">
        <input
          type="checkbox"
          checked={value.enabled}
          onChange={(e) => updateField('enabled', e.target.checked)}
        />
        <span>Enable HTTPS</span>
      </label>

      {value.enabled && (
        <>
          <label className="form-field">
            <span>Key File Path</span>
            <input
              type="text"
              value={value.keyFile}
              onChange={(e) => updateField('keyFile', e.target.value)}
              placeholder="/path/to/key.pem"
            />
            {errors['https.keyFile'] && <p className="field-error">{errors['https.keyFile']}</p>}
          </label>

          <label className="form-field">
            <span>Certificate File Path</span>
            <input
              type="text"
              value={value.certFile}
              onChange={(e) => updateField('certFile', e.target.value)}
              placeholder="/path/to/cert.pem"
            />
            {errors['https.certFile'] && (
              <p className="field-error">{errors['https.certFile']}</p>
            )}
          </label>

          <label className="form-field">
            <span>CA File Path (optional)</span>
            <input
              type="text"
              value={value.caFile}
              onChange={(e) => updateField('caFile', e.target.value)}
              placeholder="/path/to/ca.pem"
            />
            {errors['https.caFile'] && <p className="field-error">{errors['https.caFile']}</p>}
          </label>
        </>
      )}
    </article>
  )
}
