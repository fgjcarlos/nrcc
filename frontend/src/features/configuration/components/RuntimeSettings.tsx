import { FileText, Activity, Globe } from 'lucide-react';
import { InputField, ToggleField, SelectField } from './FormFields';
import type { NodeRedConfigFormData } from '@/shared/types';

interface RuntimeSettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
}

export function RuntimeSettings({ settings, onUpdate, disabled }: RuntimeSettingsProps) {
  return (
    <div className="space-y-6">
      {/* Files Section */}
      <div className="space-y-4">
        <div className="flex items-center gap-2 mb-4">
          <FileText className="w-5 h-5 text-base-content/60" />
          <h3 className="text-lg font-medium text-base-content">Files</h3>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <InputField
            label="Flow File"
            value={settings.flowFile}
            onChange={(v) => onUpdate('flowFile', v as string)}
            placeholder="flows.json"
            help="File to store the flows"
            disabled={disabled}
          />
          <InputField
            label="User Directory"
            value={settings.userDir}
            onChange={(v) => onUpdate('userDir', v as string)}
            placeholder="$HOME/.node-red"
            help="Directory for user data"
            disabled={disabled}
          />
          <InputField
            label="Nodes Directory"
            value={settings.nodesDir}
            onChange={(v) => onUpdate('nodesDir', v as string)}
            placeholder="$HOME/.node-red/nodes"
            help="Additional directory for installed nodes"
            disabled={disabled}
          />
        </div>
      </div>

      {/* Projects Section */}
      <div className="space-y-4">
        <div className="flex items-center gap-2 mb-4">
          <FileText className="w-5 h-5 text-base-content/60" />
          <h3 className="text-lg font-medium text-base-content">Projects</h3>
        </div>
        
        <ToggleField
          label="Enable Projects"
          value={settings.projectsEnabled}
          onChange={(v) => onUpdate('projectsEnabled', v)}
          help="Enable the projects feature for flow versioning"
          disabled={disabled}
        />
      </div>

      {/* Runtime State Section */}
      <div className="space-y-4">
        <div className="flex items-center gap-2 mb-4">
          <Activity className="w-5 h-5 text-base-content/60" />
          <h3 className="text-lg font-medium text-base-content">Runtime State</h3>
        </div>
        
        <div className="space-y-3">
          <ToggleField
            label="Enable Runtime State"
            value={settings.runtimeStateEnabled}
            onChange={(v) => onUpdate('runtimeStateEnabled', v)}
            help="Persist runtime state between restarts"
            disabled={disabled}
          />
          {settings.runtimeStateEnabled && (
            <InputField
              label="State File"
              value={settings.runtimeStateFile}
              onChange={(v) => onUpdate('runtimeStateFile', v as string)}
              placeholder="runtime-state.json"
              disabled={disabled}
            />
          )}
        </div>
      </div>

      {/* Language Section */}
      <div className="space-y-4">
        <div className="flex items-center gap-2 mb-4">
          <Globe className="w-5 h-5 text-base-content/60" />
          <h3 className="text-lg font-medium text-base-content">Language</h3>
        </div>
        
        <SelectField
          label="Runtime Language"
          value={settings.lang}
          onChange={(v) => onUpdate('lang', v)}
          options={[
            { value: 'en-US', label: 'English (US)' },
            { value: 'en', label: 'English' },
            { value: 'es', label: 'Español' },
            { value: 'de', label: 'Deutsch' },
            { value: 'fr', label: 'Français' },
            { value: 'ja', label: '日本語' },
            { value: 'zh-CN', label: '简体中文' },
            { value: 'zh-TW', label: '繁體中文' },
          ]}
          disabled={disabled}
        />
      </div>
    </div>
  );
}
