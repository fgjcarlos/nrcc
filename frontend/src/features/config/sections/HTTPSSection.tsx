import { HTTPSConfig } from '../../../types/config'
import { FormField } from '../../../components/forms'

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
    <article className="surface-card border border-base-300/60 p-6 md:p-7 space-y-6">
      <div className="config-section-head">
        <p className="config-section-kicker">Transport</p>
        <h3 className="config-section-title">HTTPS</h3>
        <p className="config-section-copy">
          Enable TLS for the runtime and point Node-RED to the key, certificate, and optional CA bundle files.
        </p>
      </div>

      {value.enabled && (
        <section className="alert alert-warning">
          <strong>HTTPS Enabled</strong>
          <p className="text-sm">
            After enabling HTTPS, you must restart Node-RED and update your browser URL to https://
          </p>
        </section>
      )}

      <label className="config-toggle-row cursor-pointer">
          <input
            type="checkbox"
            className="checkbox"
            checked={value.enabled}
            onChange={(e) => updateField('enabled', e.target.checked)}
          />
          <span className="config-toggle-copy">
            <span className="config-toggle-title">Enable HTTPS</span>
            <span className="config-toggle-hint">Serve the editor and HTTP endpoints over TLS after the next restart.</span>
          </span>
      </label>

        {value.enabled && (
         <div className="config-subsection space-y-6">
          <div>
            <p className="config-subsection-title">Certificate files</p>
            <p className="config-subsection-copy">Use absolute paths accessible from inside the Node-RED container or runtime host.</p>
          </div>
          <FormField
            id="https-key-file"
            label="Key File Path"
            type="text"
            value={value.keyFile}
            onChange={(v) => updateField('keyFile', v)}
            placeholder="/path/to/key.pem"
            error={errors['https.keyFile']}
          />

          <FormField
            id="https-cert-file"
            label="Certificate File Path"
            type="text"
            value={value.certFile}
            onChange={(v) => updateField('certFile', v)}
            placeholder="/path/to/cert.pem"
            error={errors['https.certFile']}
          />

          <FormField
            id="https-ca-file"
            label="CA File Path (optional)"
            type="text"
            value={value.caFile}
            onChange={(v) => updateField('caFile', v)}
            placeholder="/path/to/ca.pem"
            error={errors['https.caFile']}
          />
         </div>
        )}
    </article>
  )
}
