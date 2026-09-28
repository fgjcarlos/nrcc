import { Shield } from 'lucide-react';
import { InputField, ToggleField } from './FormFields';
import { FieldStatusChip } from './FieldStatusChip';
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
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 mb-4">
        <Shield className="w-5 h-5 text-base-content/60" />
        <h3 className="text-lg font-medium text-base-content">{t('configuration:security')}</h3>
      </div>

      <div>
        <h4 className="mb-3 text-sm font-medium text-base-content/60">{t('configuration:credentialEncryption')}</h4>
        <div className="space-y-1">
          <InputField
            label="Credential Secret"
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
        <h4 className="mb-3 text-sm font-medium text-base-content/60">{t('configuration:httpsRedirect')}</h4>
        <div className="space-y-1">
          <ToggleField
            label="Require HTTPS"
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
        <h4 className="mb-3 text-sm font-medium text-base-content/60">{t('configuration:tlsHttpsListener')}</h4>
        <p className="mb-3 text-xs text-base-content/60">
          On-disk paths to PEM-encoded files. Node-RED reads each entry through fs.readFileSync at startup; the file must remain readable by the Node-RED process and Node-RED must be restarted for changes to take effect.
        </p>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="space-y-1">
            <InputField
              label="Private Key Path"
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
    </div>
  );
}
