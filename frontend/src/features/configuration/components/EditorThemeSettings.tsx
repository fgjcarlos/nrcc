import { InputField, ToggleField, SelectField } from './FormFields';
import { ImageUpload } from './ImageUpload';
import type { NodeRedConfigFormData } from '@/shared/types';
import { useT } from '@/i18n';

interface EditorThemeSettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
}

export function EditorThemeSettings({ settings, onUpdate, disabled }: EditorThemeSettingsProps) {
  const { t } = useT();
  return (
    <div className="space-y-4">
      <div className="space-y-6">
        {/* Page Settings */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:editorPage')}</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <InputField
              label="Page Title"
              field="editorPageTitle"
              value={settings.editorPageTitle}
              onChange={(v) => onUpdate('editorPageTitle', v as string)}
              placeholder="Node-RED"
              disabled={disabled}
            />
            <div>
              <ImageUpload
                label="Favicon"
                field="editorPageFavicon"
                value={settings.editorPageFavicon}
                onChange={(v) => onUpdate('editorPageFavicon', v)}
                type="favicon"
                help="Recommended: 32x32 or 64x64 .ico"
              />
            </div>
          </div>
        </div>

        {/* Header Settings */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">Header</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <InputField
              label="Header Title"
              field="editorHeaderTitle"
              value={settings.editorHeaderTitle}
              onChange={(v) => onUpdate('editorHeaderTitle', v as string)}
              placeholder="Node-RED"
              disabled={disabled}
            />
            <InputField
              label="Header URL"
              field="editorHeaderUrl"
              value={settings.editorHeaderUrl}
              onChange={(v) => onUpdate('editorHeaderUrl', v as string)}
              placeholder="http://nodered.org"
              disabled={disabled}
            />
            <div className="md:col-span-2">
              <ImageUpload
                label="Header Image"
                field="editorHeaderImage"
                value={settings.editorHeaderImage}
                onChange={(v) => onUpdate('editorHeaderImage', v)}
                type="header"
                help="Recommended: 150x50 PNG/JPG"
              />
            </div>
          </div>
        </div>

        {/* Palette */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">Palette</h4>
          <div className="space-y-3">
            <ToggleField
              label="Allow Node Installation"
              field="editorPaletteEditable"
              value={settings.editorPaletteEditable}
              onChange={(v) => onUpdate('editorPaletteEditable', v)}
              help="Allow users to install nodes from the palette"
              disabled={disabled}
            />
            <InputField
              label="Node Catalogues (one per line)"
              field="editorPaletteCatalogues"
              value={settings.editorPaletteCatalogues}
              onChange={(v) => onUpdate('editorPaletteCatalogues', v as string)}
              placeholder="https://catalogue.nodered.org/catalogue.json"
              disabled={disabled}
            />
          </div>
        </div>

        {/* Code Editor */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:codeEditor')}</h4>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <SelectField
              label="Editor Library"
              field="editorCodeLib"
              value={settings.editorCodeLib}
              onChange={(v) => onUpdate('editorCodeLib', v)}
              options={[
                { value: 'ace', label: 'Ace' },
                { value: 'monaco', label: 'Monaco' },
              ]}
              disabled={disabled}
            />
            <SelectField
              label="Theme"
              field="editorCodeTheme"
              value={settings.editorCodeTheme}
              onChange={(v) => onUpdate('editorCodeTheme', v)}
              options={[
                { value: 'vs', label: 'Light (vs)' },
                { value: 'vs-dark', label: 'Dark (vs-dark)' },
                { value: 'hc-black', label: 'High Contrast' },
              ]}
              disabled={disabled}
            />
            <InputField
              label="Font Size"
              field="editorCodeFontSize"
              value={settings.editorCodeFontSize}
              onChange={(v) => onUpdate('editorCodeFontSize', v as number)}
              type="number"
              disabled={disabled}
            />
          </div>
        </div>

        {/* Misc */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">Miscellaneous</h4>
          <div className="space-y-3">
            <ToggleField
              label="Show User Menu"
              field="editorUserMenu"
              value={settings.editorUserMenu}
              onChange={(v) => onUpdate('editorUserMenu', v)}
              disabled={disabled}
            />
            <ToggleField
              label="Enable Welcome Tours"
              field="editorTours"
              value={settings.editorTours}
              onChange={(v) => onUpdate('editorTours', v)}
              disabled={disabled}
            />
            <ToggleField
              label="Enable Projects in Editor"
              field="editorProjectsEnabled"
              value={settings.editorProjectsEnabled}
              onChange={(v) => onUpdate('editorProjectsEnabled', v)}
              disabled={disabled}
            />
          </div>
        </div>

        {/* Login Image */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:loginScreen')}</h4>
          <ImageUpload
            label="Login Background Image"
            field="editorLoginImage"
            value={settings.editorLoginImage}
            onChange={(v) => onUpdate('editorLoginImage', v)}
            type="login"
            help="Recommended: 1920x1080 PNG/JPG"
          />
          <div className="mt-3">
            <InputField
              label="Logout Redirect URL"
              field="editorLogoutRedirect"
              value={settings.editorLogoutRedirect}
              onChange={(v) => onUpdate('editorLogoutRedirect', v as string)}
              placeholder="http://example.com"
              help="URL to redirect after logout"
              disabled={disabled}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
