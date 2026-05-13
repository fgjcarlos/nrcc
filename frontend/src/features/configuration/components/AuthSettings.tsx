import { Lock } from 'lucide-react';
import { InputField, ToggleField } from './FormFields';
import type { NodeRedConfigFormData } from '@/shared/types';

interface AuthSettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
}

export function AuthSettings({ settings, onUpdate, disabled }: AuthSettingsProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-4">
        <Lock className="w-5 h-5 text-base-content/60" />
        <h3 className="text-lg font-medium text-base-content">Authentication</h3>
      </div>
      
      <div className="space-y-6">
        {/* Admin Auth */}
        <div>
          <h4 className="mb-3 text-sm font-medium text-base-content/60">Admin Authentication</h4>
          <div className="space-y-3">
            <ToggleField
              label="Enable Admin Auth"
              value={settings.authEnabled}
              onChange={(v) => onUpdate('authEnabled', v)}
              help="Require login to access the editor"
              disabled={disabled}
            />
            {settings.authEnabled && (
              <div className="grid grid-cols-1 gap-4 border-l border-border pl-4 md:grid-cols-2">
                <div>
                  <InputField
                    label="Username"
                    value={settings.authAdminUser}
                    onChange={(v) => onUpdate('authAdminUser', v as string)}
                    placeholder="admin"
                    help={settings.authAdminUser && settings.authAdminUser.length < 3 ? 'Min 3 characters' : undefined}
                    disabled={disabled}
                  />
                </div>
                <div>
                  <InputField
                    label="Password"
                    value={settings.authAdminPassword}
                    onChange={(v) => onUpdate('authAdminPassword', v as string)}
                    type="password"
                    placeholder="Enter new password"
                    help={settings.authAdminPassword && settings.authAdminPassword.length < 6 ? 'Min 6 characters' : undefined}
                    disabled={disabled}
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Node HTTP Auth */}
        <div>
          <h4 className="mb-3 text-sm font-medium text-base-content/60">Node HTTP Authentication</h4>
          <div className="space-y-3">
            <ToggleField
              label="Enable Node HTTP Auth"
              value={settings.authNodeHttpEnabled}
              onChange={(v) => onUpdate('authNodeHttpEnabled', v)}
              help="Require login for HTTP nodes (http request, etc)"
              disabled={disabled}
            />
            {settings.authNodeHttpEnabled && (
              <div className="grid grid-cols-1 gap-4 border-l border-border pl-4 md:grid-cols-2">
                <div>
                  <InputField
                    label="Username"
                    value={settings.authNodeHttpUser}
                    onChange={(v) => onUpdate('authNodeHttpUser', v as string)}
                    placeholder="user"
                    help={settings.authNodeHttpUser && settings.authNodeHttpUser.length < 3 ? 'Min 3 characters' : undefined}
                    disabled={disabled}
                  />
                </div>
                <div>
                  <InputField
                    label="Password"
                    value={settings.authNodeHttpPassword}
                    onChange={(v) => onUpdate('authNodeHttpPassword', v as string)}
                    type="password"
                    placeholder="Enter new password"
                    help={settings.authNodeHttpPassword && settings.authNodeHttpPassword.length < 6 ? 'Min 6 characters' : undefined}
                    disabled={disabled}
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Static Auth */}
        <div>
          <h4 className="mb-3 text-sm font-medium text-base-content/60">Static Authentication</h4>
          <div className="space-y-3">
            <ToggleField
              label="Enable Static Auth"
              value={settings.authStaticEnabled}
              onChange={(v) => onUpdate('authStaticEnabled', v)}
              help="Basic auth for static HTTP paths"
              disabled={disabled}
            />
            {settings.authStaticEnabled && (
              <div className="grid grid-cols-1 gap-4 border-l border-border pl-4 md:grid-cols-2">
                <div>
                  <InputField
                    label="Username"
                    value={settings.authStaticUser}
                    onChange={(v) => onUpdate('authStaticUser', v as string)}
                    placeholder="user"
                    help={settings.authStaticUser && settings.authStaticUser.length < 3 ? 'Min 3 characters' : undefined}
                    disabled={disabled}
                  />
                </div>
                <div>
                  <InputField
                    label="Password"
                    value={settings.authStaticPassword}
                    onChange={(v) => onUpdate('authStaticPassword', v as string)}
                    type="password"
                    placeholder="Enter new password"
                    help={settings.authStaticPassword && settings.authStaticPassword.length < 6 ? 'Min 6 characters' : undefined}
                    disabled={disabled}
                  />
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
