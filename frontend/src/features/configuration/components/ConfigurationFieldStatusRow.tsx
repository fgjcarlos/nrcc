import { useT } from '@/i18n';
import type { FieldDiff } from '../hooks/useConfigurationDiff';
import type { NodeRedConfigFormData } from '@/shared/types';

interface ConfigurationFieldStatusRowProps {
  fieldKey: keyof NodeRedConfigFormData;
  label: string;
  diff: FieldDiff;
}

/**
 * Issue #766 slice F — six-column row widget for the
 * Configuration six-element header.
 *
 * Columns:
 *   configured value | effective value | source | validation |
 *   pending change   | restart required
 *
 * The row is intentionally a `<tr>` so it composes inside the
 * existing `<table>` structure that `ConfigurationView` uses for the
 * per-tab form.
 */
export function ConfigurationFieldStatusRow({
  fieldKey,
  label,
  diff,
}: ConfigurationFieldStatusRowProps) {
  const { t } = useT();

  return (
    <tr data-field-key={fieldKey} data-pending={diff.pending ? 'true' : 'false'}>
      <th scope="row" className="text-left align-top text-sm font-medium text-base-content/85">
        {label}
      </th>
      <td className="text-sm text-base-content" data-testid="configuration-field-status-configured">
        {formatValue(diff.configuredValue)}
      </td>
      <td className="text-sm text-base-content/75" data-testid="configuration-field-status-effective">
        <span className="block text-xs text-base-content/50">{diff.effectiveLabel}</span>
        {formatValue(diff.effectiveValue)}
      </td>
      <td className="text-xs uppercase tracking-wide text-base-content/55">
        {t(`configuration:fieldStatus.source.${diff.source === 'settings.js' ? 'settingsJs' : 'formDefault'}`)}
      </td>
      <td data-testid="configuration-field-status-validation" data-state={diff.validation.state}>
        {diff.validation.state === 'invalid' && diff.validation.message ? (
          <span className="text-warning">{diff.validation.message}</span>
        ) : diff.validation.state === 'pending' ? (
          <span className="text-base-content/55">{t('configuration:fieldStatus.validation.pending')}</span>
        ) : (
          <span className="text-success">{t('configuration:fieldStatus.validation.valid')}</span>
        )}
      </td>
      <td data-testid="configuration-field-status-pending" data-pending={diff.pending ? 'true' : 'false'}>
        {diff.pending ? (
          <span className="rounded-full border border-primary/40 px-2 py-0.5 text-xs text-primary">
            {t('configuration:fieldStatus.pending.changed')}
          </span>
        ) : (
          <span className="text-base-content/40">—</span>
        )}
      </td>
      <td data-testid="configuration-field-status-restart" data-severity={diff.restart}>
        {diff.restart === 'hard' ? (
          <span className="rounded-full border border-error/40 px-2 py-0.5 text-xs text-error">
            {t('configuration:fieldStatus.restart.hard')}
          </span>
        ) : diff.restart === 'soft' ? (
          <span className="rounded-full border border-warning/40 px-2 py-0.5 text-xs text-warning">
            {t('configuration:fieldStatus.restart.soft')}
          </span>
        ) : (
          <span className="text-base-content/40">—</span>
        )}
      </td>
    </tr>
  );
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  if (typeof value === 'string' && value.length === 0) return '—';
  if (typeof value === 'string' && value.length > 80) return `${value.slice(0, 77)}…`;
  return String(value);
}