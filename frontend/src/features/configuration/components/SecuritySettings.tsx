import { Shield } from 'lucide-react';
import { InputField, ToggleField } from './FormFields';
import type { NodeRedConfigFormData } from '@/shared/types';

interface SecuritySettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
}

// Issue #762. credentialSecret rotation, TLS `https` block, and
// requireHttps redirect (PR #776). Rotation confirmation lives in
// ConfigurationView (save time). Hidden when editable=false.
export function SecuritySettings({ settings, onUpdate, disabled }: SecuritySettingsProps) {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 mb-4">
        <Shield className="w-5 h-5 text-base-content/60" />
        <h3 className="text-lg font-medium text-base-content">Security</h3>
      </div>

      <div>
        <h4 className="mb-3 text-sm font-medium text-base-content/60">Credential Encryption</h4>
        <InputField
          label="Credential Secret"
          value={settings.credentialSecret ?? ''}
          onChange={(v) => onUpdate('credentialSecret', v as string)}
          type="password"
          placeholder="Leave blank to keep current value"
          help="Leave blank to keep the current value. Enter a new passphrase to rotate credentials — existing encrypted credentials must be re-entered after a rotation, and Node-RED must be restarted."
          disabled={disabled}
        />
      </div>

      <div>
        <h4 className="mb-3 text-sm font-medium text-base-content/60">HTTPS Redirect</h4>
        <ToggleField
          label="Require HTTPS"
          value={settings.requireHttps ?? false}
          onChange={(v) => onUpdate('requireHttps', v)}
          help="Redirect plain HTTP traffic to the HTTPS listener. Requires the TLS block below and a Node-RED restart."
          disabled={disabled}
        />
      </div>

      {/* TLS settings. Each input is an on-disk path rendered as
          fs.readFileSync(<path>) — certificate bytes never embedded. */}
      <div>
        <h4 className="mb-3 text-sm font-medium text-base-content/60">TLS (HTTPS Listener)</h4>
        <p className="mb-3 text-xs text-base-content/60">
          On-disk paths to PEM-encoded files. Node-RED reads each entry through fs.readFileSync at startup; the file must remain readable by the Node-RED process and Node-RED must be restarted for changes to take effect.
        </p>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <InputField
            label="Private Key Path"
            value={settings.httpsKey ?? ''}
            onChange={(v) => onUpdate('httpsKey', v as string)}
            placeholder="/etc/node-red/key.pem"
            help="Path to the PEM-encoded private key."
            disabled={disabled}
          />
          <InputField
            label="Certificate Path"
            value={settings.httpsCert ?? ''}
            onChange={(v) => onUpdate('httpsCert', v as string)}
            placeholder="/etc/node-red/cert.pem"
            help="Path to the PEM-encoded server certificate."
            disabled={disabled}
          />
          <InputField
            label="CA Bundle Path"
            value={settings.httpsCA ?? ''}
            onChange={(v) => onUpdate('httpsCA', v as string)}
            placeholder="/etc/node-red/ca.pem"
            help="Optional path to the PEM-encoded CA bundle for client certificate verification."
            disabled={disabled}
          />
          <InputField
            label="HTTPS Port"
            value={settings.httpsPort ?? 0}
            onChange={(v) => onUpdate('httpsPort', v as number)}
            type="number"
            placeholder="1880"
            help="Port for the HTTPS listener. Defaults to uiPort when zero."
            disabled={disabled}
          />
          <div className="md:col-span-2">
            <InputField
              label="Private Key Passphrase"
              value={settings.httpsPassphrase ?? ''}
              onChange={(v) => onUpdate('httpsPassphrase', v as string)}
              type="password"
              placeholder="Leave blank for unencrypted keys"
              help="Optional passphrase for an encrypted private key. Leave blank to keep the current value."
              disabled={disabled}
            />
          </div>
        </div>
      </div>
    </div>
  );
}