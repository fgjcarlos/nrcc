import { Activity } from 'lucide-react';
import { SelectField, ToggleField, LOGGING_LEVELS } from './FormFields';
import type { NodeRedConfigFormData } from '@/shared/types';

interface LoggingSettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
}

export function LoggingSettings({ settings, onUpdate, disabled }: LoggingSettingsProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 mb-4">
        <Activity className="w-5 h-5 text-base-content/60" />
        <h3 className="text-lg font-medium text-base-content">Logging</h3>
      </div>
      
      <div className="space-y-6">
        {/* Console Handler */}
        <div>
          <h4 className="mb-3 text-sm font-medium text-base-content/60">Console Handler</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <SelectField
              label="Console Level"
              value={settings.loggingConsoleLevel}
              onChange={(v) => onUpdate('loggingConsoleLevel', v)}
              options={LOGGING_LEVELS}
              disabled={disabled}
            />
            <ToggleField
              label="Enable Metrics"
              value={settings.loggingConsoleMetrics}
              onChange={(v) => onUpdate('loggingConsoleMetrics', v)}
              help="Log flow execution metrics"
              disabled={disabled}
            />
          </div>
        </div>

        {/* Internal Handler */}
        <div>
          <h4 className="mb-3 text-sm font-medium text-base-content/60">Internal Handler</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <SelectField
              label="Internal Level"
              value={settings.loggingInternalLevel}
              onChange={(v) => onUpdate('loggingInternalLevel', v)}
              options={LOGGING_LEVELS}
              disabled={disabled}
            />
            <ToggleField
              label="Enable Metrics"
              value={settings.loggingInternalMetrics}
              onChange={(v) => onUpdate('loggingInternalMetrics', v)}
              help="Log runtime metrics"
              disabled={disabled}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
