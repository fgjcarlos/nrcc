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
    <article className="space-y-6">
      <h3 className="text-xl font-semibold text-base-content">HTTPS</h3>

      {value.enabled && (
        <section className="alert alert-warning">
          <strong>HTTPS Enabled</strong>
          <p className="text-sm">
            After enabling HTTPS, you must restart Node-RED and update your browser URL to https://
          </p>
        </section>
      )}

      <div className="form-control">
        <label className="label cursor-pointer gap-3">
          <input
            type="checkbox"
            className="checkbox checkbox-sm"
            checked={value.enabled}
            onChange={(e) => updateField('enabled', e.target.checked)}
          />
          <span className="label-text font-medium">Enable HTTPS</span>
        </label>
      </div>

       {value.enabled && (
         <div className="space-y-6 pl-4 border-l-2 border-[color:var(--border-indent)]">
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
