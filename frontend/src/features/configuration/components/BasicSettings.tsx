import { Server } from 'lucide-react';
import { InputField, ToggleField } from './FormFields';
import type { NodeRedConfigFormData } from '@/shared/types';

interface BasicSettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
}

export function BasicSettings({ settings, onUpdate, disabled }: BasicSettingsProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-4">
        <Server className="w-5 h-5 text-base-content/60" />
        <h3 className="text-lg font-medium text-base-content">Basic Settings</h3>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <InputField
          label="UI Port"
          value={settings.uiPort}
          onChange={(v) => onUpdate('uiPort', v as number)}
          type="number"
          placeholder="1880"
          help="Port for the editor UI"
          disabled={disabled}
        />
        <InputField
          label="UI Host"
          value={settings.uiHost}
          onChange={(v) => onUpdate('uiHost', v as string)}
          placeholder="0.0.0.0"
          help="Interface to listen on"
          disabled={disabled}
        />
        <InputField
          label="Admin Root"
          value={settings.httpAdminRoot}
          onChange={(v) => onUpdate('httpAdminRoot', v as string)}
          placeholder="/"
          help="Root URL for the editor"
          disabled={disabled}
        />
        <InputField
          label="Node Root"
          value={settings.httpNodeRoot}
          onChange={(v) => onUpdate('httpNodeRoot', v as string)}
          placeholder="/"
          help="Root URL for node HTTP endpoints"
          disabled={disabled}
        />
        <div className="md:col-span-2">
          <ToggleField
            label="Disable Editor"
            value={settings.disableEditor}
            onChange={(v) => onUpdate('disableEditor', v)}
            help="Prevent the editor UI from being served"
            disabled={disabled}
          />
        </div>
      </div>
    </div>
  );
}
