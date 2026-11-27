import { InputField, ToggleField, FieldLabel } from './FormFields';
import { FieldStatusChip } from './FieldStatusChip';
import { Textarea } from '@/shared/components/ui';
import { DEFAULT_HTTP_NODE_CORS_METHODS, HTTP_NODE_CORS_METHODS } from '@/shared/types';
import type { NodeRedConfigFormData } from '@/shared/types';
import type { FieldDiff } from '../hooks/useConfigurationDiff';
import { useT } from '@/i18n';

interface SecuritySettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
  /**
   * Per-field validation state, keyed by the `NodeRedConfigFormData`
   * field name. Issue #766 slice F W2 — the `FieldStatusChip` lights
   * up the credential secret, TLS path, and port fields as soon as
   * the operator types a value that fails the per-field validator.
   */
  fieldErrors?: Partial<Record<keyof NodeRedConfigFormData, FieldDiff['validation']>>;
}

// Issue #762. credentialSecret rotation, TLS `https` block, and
// requireHttps redirect (PR #776). Rotation confirmation lives in
// ConfigurationView (save time). Hidden when editable=false.
export function SecuritySettings({ settings, onUpdate, disabled, fieldErrors }: SecuritySettingsProps) {
  const { t } = useT();
  // The flat model keeps methods as a comma separated string; rewriting it
  // from the current tokens preserves any method the UI does not offer.
  const methods = (settings.httpNodeCorsMethods ?? DEFAULT_HTTP_NODE_CORS_METHODS.join(','))
    .split(',')
    .map((method) => method.trim())
    .filter(Boolean);
  const origins = (settings.httpNodeCorsOrigins ?? '').split('\n').map((origin) => origin.trim());
  const toggleMethod = (method: string) => {
    const next = methods.includes(method) ? methods.filter((entry) => entry !== method) : [...methods, method];
    onUpdate('httpNodeCorsMethods', next.join(','));
  };
  return (
    <div className="space-y-6">
      <div>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:credentialEncryption')}</h4>
        <div className="space-y-1">
          <InputField
            label="Credential Secret"
            field="credentialSecret"
            value={settings.credentialSecret ?? ''}
            onChange={(v) => onUpdate('credentialSecret', v as string)}
            type="password"
            placeholder="Leave blank to keep current value"
            help="Leave blank to keep the current value. Enter a new passphrase to rotate credentials — existing encrypted credentials must be re-entered after a rotation, and Node-RED must be restarted."
            disabled={disabled}
            error={fieldErrors?.credentialSecret?.state === 'invalid'}
          />
          <FieldStatusChip
            validation={fieldErrors?.credentialSecret ?? { state: 'valid' }}
            fieldKey="credentialSecret"
          />
        </div>
      </div>

      <div>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:httpsRedirect')}</h4>
        <div className="space-y-1">
          <ToggleField
            label="Require HTTPS"
            field="requireHttps"
            value={settings.requireHttps ?? false}
            onChange={(v) => onUpdate('requireHttps', v)}
            help="Redirect plain HTTP traffic to the HTTPS listener. Requires the TLS block below and a Node-RED restart."
            disabled={disabled}
          />
          <FieldStatusChip
            validation={fieldErrors?.requireHttps ?? { state: 'valid' }}
            fieldKey="requireHttps"
          />
        </div>
      </div>

      {/* TLS settings. Each input is an on-disk path rendered as
          fs.readFileSync(<path>) — certificate bytes never embedded. */}
      <div>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:tlsHttpsListener')}</h4>
        <p className="mb-3 text-xs text-ds-text-muted">
          On-disk paths to PEM-encoded files. Node-RED reads each entry through fs.readFileSync at startup; the file must remain readable by the Node-RED process and Node-RED must be restarted for changes to take effect.
        </p>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="space-y-1">
            <InputField
              label="Private Key Path"
              field="httpsKey"
              value={settings.httpsKey ?? ''}
              onChange={(v) => onUpdate('httpsKey', v as string)}
              placeholder="/etc/node-red/key.pem"
              help="Path to the PEM-encoded private key."
              disabled={disabled}
              error={fieldErrors?.httpsKey?.state === 'invalid'}
            />
            <FieldStatusChip validation={fieldErrors?.httpsKey ?? { state: 'valid' }} fieldKey="httpsKey" />
          </div>
          <div className="space-y-1">
            <InputField
              label="Certificate Path"
              field="httpsCert"
              value={settings.httpsCert ?? ''}
              onChange={(v) => onUpdate('httpsCert', v as string)}
              placeholder="/etc/node-red/cert.pem"
              help="Path to the PEM-encoded server certificate."
              disabled={disabled}
              error={fieldErrors?.httpsCert?.state === 'invalid'}
            />
            <FieldStatusChip validation={fieldErrors?.httpsCert ?? { state: 'valid' }} fieldKey="httpsCert" />
          </div>
          <div className="space-y-1">
            <InputField
              label="CA Bundle Path"
              field="httpsCA"
              value={settings.httpsCA ?? ''}
              onChange={(v) => onUpdate('httpsCA', v as string)}
              placeholder="/etc/node-red/ca.pem"
              help="Optional path to the PEM-encoded CA bundle for client certificate verification."
              disabled={disabled}
              error={fieldErrors?.httpsCA?.state === 'invalid'}
            />
            <FieldStatusChip validation={fieldErrors?.httpsCA ?? { state: 'valid' }} fieldKey="httpsCA" />
          </div>
          <div className="space-y-1">
            <InputField
              label="HTTPS Port"
              field="httpsPort"
              value={settings.httpsPort ?? 0}
              onChange={(v) => onUpdate('httpsPort', v as number)}
              type="number"
              placeholder="1880"
              help="Port for the HTTPS listener. Defaults to uiPort when zero."
              disabled={disabled}
              error={fieldErrors?.httpsPort?.state === 'invalid'}
            />
            <FieldStatusChip validation={fieldErrors?.httpsPort ?? { state: 'valid' }} fieldKey="httpsPort" />
          </div>
          <div className="md:col-span-2 space-y-1">
            <InputField
              label="Private Key Passphrase"
              field="httpsPassphrase"
              value={settings.httpsPassphrase ?? ''}
              onChange={(v) => onUpdate('httpsPassphrase', v as string)}
              type="password"
              placeholder="Leave blank for unencrypted keys"
              help="Optional passphrase for an encrypted private key. Leave blank to keep the current value."
              disabled={disabled}
              error={fieldErrors?.httpsPassphrase?.state === 'invalid'}
            />
            <FieldStatusChip
              validation={fieldErrors?.httpsPassphrase ?? { state: 'valid' }}
              fieldKey="httpsPassphrase"
            />
          </div>
        </div>
      </div>

      {/* CORS for http-in endpoints (issue #1076). An unsupported
          hand-written block is shown read-only: no controls, verbatim
          source, and it round-trips unchanged. */}
      <div>
        <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:httpNodeCors.title')}</h4>
        {settings.httpNodeCorsUnsupported ? (
          <div className="space-y-2">
            <p className="text-xs text-ds-text-muted">{t('configuration:httpNodeCors.unsupportedHelp')}</p>
            <pre
              aria-label={t('configuration:httpNodeCors.rawLabel')}
              className="overflow-x-auto rounded-box border border-ds-border-subtle bg-ds-bg-base p-3 font-mono text-xs text-ds-text-primary"
            >
              {settings.httpNodeCorsRaw ?? ''}
            </pre>
          </div>
        ) : (
          <div className="space-y-3">
            <p className="text-xs text-ds-text-muted">{t('configuration:httpNodeCors.description')}</p>
            <ToggleField
              label={t('configuration:httpNodeCors.enable')}
              field="httpNodeCorsEnabled"
              value={settings.httpNodeCorsEnabled ?? false}
              onChange={(v) => onUpdate('httpNodeCorsEnabled', v)}
              help={t('configuration:httpNodeCors.enableHelp')}
              disabled={disabled}
            />
            {settings.httpNodeCorsEnabled && (
              <div className="space-y-4">
                <div className="space-y-1">
                  <FieldLabel label={t('configuration:httpNodeCors.origins')} settingKey="httpNodeCors.origin" htmlFor="cfg-httpNodeCorsOrigins" />
                  <Textarea
                    id="cfg-httpNodeCorsOrigins"
                    value={settings.httpNodeCorsOrigins ?? ''}
                    onChange={(e) => onUpdate('httpNodeCorsOrigins', e.target.value)}
                    placeholder={t('configuration:httpNodeCors.originsPlaceholder')}
                    disabled={disabled}
                    rows={3}
                  />
                  <p className="text-xs text-ds-text-muted">{t('configuration:httpNodeCors.originsHelp')}</p>
                </div>

                <fieldset className="space-y-1">
                  <legend className="text-xs text-ds-text-secondary">{t('configuration:httpNodeCors.methods')}</legend>
                  <div className="flex flex-wrap gap-x-4 gap-y-1">
                    {HTTP_NODE_CORS_METHODS.map((method) => (
                      <label key={method} className="flex items-center gap-1.5 text-xs text-ds-text-secondary">
                        <input
                          type="checkbox"
                          checked={methods.includes(method)}
                          onChange={() => toggleMethod(method)}
                          disabled={disabled}
                        />
                        {method}
                      </label>
                    ))}
                  </div>
                  <p className="text-xs text-ds-text-muted">{t('configuration:httpNodeCors.methodsHelp')}</p>
                </fieldset>

                <ToggleField
                  label={t('configuration:httpNodeCors.credentials')}
                  field="httpNodeCorsCredentials"
                  value={settings.httpNodeCorsCredentials ?? false}
                  onChange={(v) => onUpdate('httpNodeCorsCredentials', v)}
                  help={t('configuration:httpNodeCors.credentialsHelp')}
                  disabled={disabled}
                />

                <InputField
                  label={t('configuration:httpNodeCors.allowedHeaders')}
                  field="httpNodeCorsAllowedHeaders"
                  value={settings.httpNodeCorsAllowedHeaders ?? ''}
                  onChange={(v) => onUpdate('httpNodeCorsAllowedHeaders', v as string)}
                  placeholder={t('configuration:httpNodeCors.allowedHeadersPlaceholder')}
                  help={t('configuration:httpNodeCors.allowedHeadersHelp')}
                  disabled={disabled}
                />

                {origins.includes('*') && (
                  <p role="alert" className="rounded-box border border-ds-warning/40 bg-ds-warning/10 px-3 py-2 text-sm text-ds-on-tint-warning">
                    {settings.httpNodeCorsCredentials
                      ? t('configuration:httpNodeCors.wildcardCredentialsWarning')
                      : t('configuration:httpNodeCors.wildcardWarning')}
                  </p>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
