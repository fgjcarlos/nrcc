import { Server } from 'lucide-react';
import { InputField, ToggleField } from './FormFields';
import { FieldStatusChip } from './FieldStatusChip';
import type { NodeRedConfigFormData } from '@/shared/types';
import type { FieldDiff } from '../hooks/useConfigurationDiff';
import { useT } from '@/i18n';

interface BasicSettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
  /**
   * Per-field validation state, keyed by the `NodeRedConfigFormData`
   * field name. Issued #766 slice F W2 — the per-field
   * `FieldStatusChip` renders next to each `InputField` /
   * `ToggleField` so the operator sees feedback while typing instead
   * of waiting for the Save-time toast.
   */
  fieldErrors?: Partial<Record<keyof NodeRedConfigFormData, FieldDiff['validation']>>;
}

export function BasicSettings({ settings, onUpdate, disabled, fieldErrors }: BasicSettingsProps) {
  const { t } = useT();
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-4">
        <Server className="w-5 h-5 text-base-content/60" />
        <h3 className="text-lg font-medium text-base-content">{t('configuration:basicSettings')}</h3>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="space-y-1">
          <InputField
            label="UI Port"
            value={settings.uiPort}
            onChange={(v) => onUpdate('uiPort', v as number)}
            type="number"
            placeholder="1880"
            help="Port for the editor UI"
            disabled={disabled}
            error={fieldErrors?.uiPort?.state === 'invalid'}
          />
          <FieldStatusChip validation={fieldErrors?.uiPort ?? { state: 'valid' }} fieldKey="uiPort" />
        </div>
        <div className="space-y-1">
          <InputField
            label="UI Host"
            value={settings.uiHost}
            onChange={(v) => onUpdate('uiHost', v as string)}
            placeholder="0.0.0.0"
            help="Interface to listen on"
            disabled={disabled}
            error={fieldErrors?.uiHost?.state === 'invalid'}
          />
          <FieldStatusChip validation={fieldErrors?.uiHost ?? { state: 'valid' }} fieldKey="uiHost" />
        </div>
        <div className="space-y-1">
          <InputField
            label="Admin Root"
            value={settings.httpAdminRoot}
            onChange={(v) => onUpdate('httpAdminRoot', v as string)}
            placeholder="/"
            help="Root URL for the editor"
            disabled={disabled}
            error={fieldErrors?.httpAdminRoot?.state === 'invalid'}
          />
          <FieldStatusChip validation={fieldErrors?.httpAdminRoot ?? { state: 'valid' }} fieldKey="httpAdminRoot" />
        </div>
        <div className="space-y-1">
          <InputField
            label="Node Root"
            value={settings.httpNodeRoot}
            onChange={(v) => onUpdate('httpNodeRoot', v as string)}
            placeholder="/"
            help="Root URL for node HTTP endpoints"
            disabled={disabled}
            error={fieldErrors?.httpNodeRoot?.state === 'invalid'}
          />
          <FieldStatusChip validation={fieldErrors?.httpNodeRoot ?? { state: 'valid' }} fieldKey="httpNodeRoot" />
        </div>
        <div className="md:col-span-2 space-y-1">
          <ToggleField
            label="Disable Editor"
            value={settings.disableEditor}
            onChange={(v) => onUpdate('disableEditor', v)}
            help="Prevent the editor UI from being served"
            disabled={disabled}
          />
          <FieldStatusChip
            validation={fieldErrors?.disableEditor ?? { state: 'valid' }}
            fieldKey="disableEditor"
          />
        </div>
      </div>
    </div>
  );
}
