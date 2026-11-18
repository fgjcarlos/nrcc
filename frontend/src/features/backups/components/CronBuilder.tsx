import { useState, useEffect, useMemo } from 'react';
import { validateCron } from '@/features/backups/lib/cronUtils';
import { useT } from '@/i18n';
import { Button, Input, Select } from '@/shared/components/ui';
import { PRESET_CRON, cronFromDateTime, dateTimeFromCron } from './cronBuilder.helpers';

const PRESETS = ['disabled', 'hourly', 'every6h', 'daily', 'weekly', 'custom'] as const;

/** Catalog key of the current validation message, or '' when valid. */
type ValidationKey = '' | 'invalidCron' | 'pickDateTime';

export type SaveState = 'idle' | 'saving' | 'saved' | 'error';
export type PresetType = 'disabled' | 'hourly' | 'every6h' | 'daily' | 'weekly' | 'custom';

export interface CronBuilderProps {
  schedule: PresetType;
  customSchedule: string;
  onChange: (cron: string) => void;
  onPresetChange: (preset: PresetType) => void;
  onSave?: () => void;
  saveState?: SaveState;
  saveError?: string;
}

export function CronBuilder({
  schedule,
  customSchedule,
  onChange,
  onPresetChange,
  onSave,
  saveState = 'idle',
  saveError,
}: CronBuilderProps) {
  const [customDate, setCustomDate] = useState('');
  const [customTime, setCustomTime] = useState('');
  const [showRawCron, setShowRawCron] = useState(false);
  const [rawCron, setRawCron] = useState('');
  const [validationError, setValidationError] = useState<ValidationKey>('');

  // ponytail: derive the active cron expression from either the
  // picker (when schedule === 'custom') or the preset map. The
  // picker's cron is computed; the parent's customSchedule is the
  // source of truth while editing.
  const { t } = useT();

  const activeCron = useMemo(() => {
    if (schedule !== 'custom') return PRESET_CRON[schedule] ?? '';
    return cronFromDateTime(customDate, customTime) ?? '';
  }, [schedule, customDate, customTime]);

  // Sync the pickers / raw cron when the parent state changes
  // (initial load, or external reset).
  useEffect(() => {
    if (schedule !== 'custom') {
      setValidationError('');
      return;
    }
    const dt = dateTimeFromCron(customSchedule);
    if (dt) {
      setCustomDate(dt.date);
      setCustomTime(dt.time);
      setRawCron(customSchedule);
      setValidationError('');
      return;
    }
    // customSchedule is not a one-shot (power user territory).
    // Fall back to raw cron so the toggle shows something useful.
    setCustomDate('');
    setCustomTime('');
    setRawCron(customSchedule);
    setValidationError(customSchedule && !validateCron(customSchedule) ? 'invalidCron' : '');
  }, [schedule, customSchedule]);

  const handlePresetSelect = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const next = e.target.value as PresetType;
    setValidationError('');
    setShowRawCron(false);
    if (next === 'custom') {
      // Initialize the picker to "now + 1h" so the operator gets a
      // sensible default rather than an empty form.
      const t = new Date();
      t.setHours(t.getHours() + 1, 0, 0, 0);
      const y = t.getFullYear();
      const mo = String(t.getMonth() + 1).padStart(2, '0');
      const d = String(t.getDate()).padStart(2, '0');
      const h = String(t.getHours()).padStart(2, '0');
      const mi = String(t.getMinutes()).padStart(2, '0');
      const initDate = `${y}-${mo}-${d}`;
      const initTime = `${h}:${mi}`;
      setCustomDate(initDate);
      setCustomTime(initTime);
      setRawCron(`${mi} ${h} ${d} ${mo} *`);
      onChange(`${mi} ${h} ${d} ${mo} *`);
      onPresetChange('custom');
    } else {
      onPresetChange(next);
      if (next !== 'disabled') onChange(PRESET_CRON[next]);
    }
  };

  const handlePickerChange = (next: { date: string; time: string }) => {
    setCustomDate(next.date);
    setCustomTime(next.time);
    const cron = cronFromDateTime(next.date, next.time);
    if (cron) {
      setRawCron(cron);
      setValidationError('');
      onChange(cron);
    } else {
      setValidationError('pickDateTime');
    }
  };

  const handleRawCronBlur = () => {
    const trimmed = rawCron.trim();
    if (!trimmed) return;
    if (validateCron(trimmed)) {
      setValidationError('');
      onChange(trimmed);
      // Try to reflect the raw cron in the pickers; if it is
      // not a one-shot, the picker goes blank (which is fine).
      const dt = dateTimeFromCron(trimmed);
      if (dt) {
        setCustomDate(dt.date);
        setCustomTime(dt.time);
      }
    } else {
      setValidationError('invalidCron');
    }
  };

  const validationMessage = validationError ? t(`backups:scheduler.${validationError}`) : '';

  return (
    <div className="space-y-4">
      <Select
        id="backup-schedule-frequency"
        label={t('backups:scheduler.frequency')}
        data-testid="preset-select"
        value={schedule}
        onChange={handlePresetSelect}
      >
        {PRESETS.map((key) => (
          <option key={key} value={key} data-testid={`preset-${key}`}>
            {t(`backups:scheduler.presets.${key}`)}
          </option>
        ))}
      </Select>

      {schedule !== 'disabled' && activeCron && (
        <p className="text-xs text-ds-text-secondary">
          {schedule === 'custom' && (
            <span className="mr-2">{t('backups:scheduler.runsOnce', { date: customDate, time: customTime })}</span>
          )}
          <span className="whitespace-nowrap">
            {t('backups:scheduler.cron')}{' '}
            <code className="font-mono font-semibold text-ds-text-primary">{activeCron}</code>
          </span>
        </p>
      )}

      {schedule === 'custom' && (
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <Input
              id="backup-schedule-date"
              label={t('backups:scheduler.date')}
              type="date"
              data-testid="custom-date"
              value={customDate}
              onChange={(e) => handlePickerChange({ date: e.target.value, time: customTime })}
              error={validationError === 'pickDateTime'}
            />
            <Input
              id="backup-schedule-time"
              label={t('backups:scheduler.time')}
              type="time"
              data-testid="custom-time"
              value={customTime}
              onChange={(e) => handlePickerChange({ date: customDate, time: e.target.value })}
              error={validationError === 'pickDateTime'}
            />
          </div>

          <label className="flex min-h-11 cursor-pointer items-center gap-2 text-sm text-ds-text-secondary md:min-h-6">
            <input
              type="checkbox"
              data-testid="custom-advanced-toggle"
              checked={showRawCron}
              onChange={(e) => setShowRawCron(e.target.checked)}
              className="checkbox checkbox-sm"
            />
            {t('backups:scheduler.advanced')}
          </label>

          {showRawCron && (
            <div className="space-y-1">
              <Input
                id="backup-schedule-cron"
                label={t('backups:scheduler.cronExpression')}
                type="text"
                data-testid="custom-cron-input"
                value={rawCron}
                onChange={(e) => setRawCron(e.target.value)}
                onBlur={handleRawCronBlur}
                placeholder="0 2 * * *"
                spellCheck={false}
                autoComplete="off"
                aria-describedby="backup-schedule-cron-hint"
                error={validationError === 'invalidCron'}
                className="font-mono"
              />
              <p id="backup-schedule-cron-hint" className="text-xs text-ds-text-secondary">
                {t('backups:scheduler.cronHint')}
              </p>
            </div>
          )}

          {validationMessage && (
            <p role="alert" className="text-xs text-ds-on-tint-error">
              {validationMessage}
            </p>
          )}
        </div>
      )}

      {/* Save button + status indicator */}
      {onSave && (
        <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
          <p role="status" className="min-w-0 text-xs">
            {saveState === 'saving' && <span className="text-ds-text-secondary">{t('common:saving')}</span>}
            {saveState === 'saved' && <span className="text-ds-on-tint-success">{t('common:saved')}</span>}
            {saveState === 'error' && saveError && <span className="text-ds-on-tint-error">{saveError}</span>}
          </p>
          <Button
            type="button"
            onClick={onSave}
            disabled={saveState === 'saving'}
            aria-busy={saveState === 'saving'}
            className="ml-auto"
            data-testid="cron-save-button"
          >
            {t('backups:scheduler.save')}
          </Button>
        </div>
      )}
    </div>
  );
}
