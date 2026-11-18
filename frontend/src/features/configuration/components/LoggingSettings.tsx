import { SelectField, ToggleField, LOGGING_LEVELS } from './FormFields';
import type { NodeRedConfigFormData } from '@/shared/types';
import { useT } from '@/i18n';

interface LoggingSettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
}

export function LoggingSettings({ settings, onUpdate, disabled }: LoggingSettingsProps) {
  const { t } = useT();
  return (
    <div className="space-y-4">
      <div className="space-y-6">
        {/* Console Handler */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:consoleHandler')}</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <SelectField
              label="Console Level"
              field="loggingConsoleLevel"
              value={settings.loggingConsoleLevel}
              onChange={(v) => onUpdate('loggingConsoleLevel', v)}
              options={LOGGING_LEVELS}
              disabled={disabled}
            />
            <ToggleField
              label="Enable Metrics"
              field="loggingConsoleMetrics"
              value={settings.loggingConsoleMetrics}
              onChange={(v) => onUpdate('loggingConsoleMetrics', v)}
              help="Log flow execution metrics"
              disabled={disabled}
            />
          </div>
        </div>

        {/* Internal Handler */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:internalHandler')}</h4>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <SelectField
              label="Internal Level"
              field="loggingInternalLevel"
              value={settings.loggingInternalLevel}
              onChange={(v) => onUpdate('loggingInternalLevel', v)}
              options={LOGGING_LEVELS}
              disabled={disabled}
            />
            <ToggleField
              label="Enable Metrics"
              field="loggingInternalMetrics"
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
