import { SelectField, ToggleField, LOGGING_LEVELS } from './FormFields';
import type { NodeRedConfigFormData } from '@/shared/types';
import { useT } from '@/i18n';

interface LoggingSettingsProps {
  settings: NodeRedConfigFormData;
  onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
  disabled?: boolean;
  configuredConsoleLevel?: string;
  runningConsoleLevel?: string;
  runtimeConsoleLoggingStatus?: 'applied' | 'pending' | 'unknown';
}

export function LoggingSettings({ settings, onUpdate, disabled, configuredConsoleLevel, runningConsoleLevel, runtimeConsoleLoggingStatus = 'unknown' }: LoggingSettingsProps) {
  const { t } = useT();
  return (
    <div className="space-y-4">
      <div className="space-y-6">
        {/* Console Handler */}
        <div>
          <h4 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ds-text-secondary">{t('configuration:consoleHandler')}</h4>
          <p className="mb-2 text-xs text-ds-text-muted">{configuredConsoleLevel ? `${t('configuration:configuredLevel')}: ${configuredConsoleLevel}` : t('configuration:levelNotSet')}</p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <SelectField
              label="Console Level"
              field="loggingConsoleLevel"
              value={settings.loggingConsoleLevel}
              onChange={(v) => onUpdate('loggingConsoleLevel', v)}
              options={[...(!settings.loggingConsoleLevel ? [{ value: '', label: t('configuration:levelNotSet'), disabled: true }] : []), ...LOGGING_LEVELS]}
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
            <ToggleField
              label={t('configuration:enableAudit')}
              field="loggingConsoleAudit"
              value={settings.loggingConsoleAudit ?? false}
              onChange={(v) => onUpdate('loggingConsoleAudit', v)}
              help={t('configuration:consoleAuditHelp')}
              disabled={disabled}
            />
          </div>
          <p className="mt-2 text-xs text-ds-text-muted">{t('configuration:restartRequired')}</p>
          <p className="mt-1 text-xs text-ds-text-muted" role="status">
            {runtimeConsoleLoggingStatus === 'applied' && runningConsoleLevel
              ? t('configuration:runtimeLevelApplied', { level: runningConsoleLevel })
              : runtimeConsoleLoggingStatus === 'pending' && runningConsoleLevel
                ? t('configuration:runtimeLevelPending', { configured: configuredConsoleLevel ?? t('configuration:levelNotSet'), running: runningConsoleLevel })
                : t('configuration:runtimeLevelUnknown')}
          </p>
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
              options={[...(!settings.loggingInternalLevel ? [{ value: '', label: t('configuration:levelNotSet'), disabled: true }] : []), ...LOGGING_LEVELS]}
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
            <ToggleField
              label={t('configuration:enableAudit')}
              field="loggingInternalAudit"
              value={settings.loggingInternalAudit ?? false}
              onChange={(v) => onUpdate('loggingInternalAudit', v)}
              help={t('configuration:internalAuditHelp')}
              disabled={disabled}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
