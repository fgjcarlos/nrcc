import { useState, useEffect, useMemo } from 'react';
import { validateCron } from '@/features/backups/lib/cronUtils';
import { UI_COPY } from '@/shared/constants/uiCopy';

const PRESET_CRON: Record<string, string> = {
  hourly: '0 * * * *',
  every6h: '0 */6 * * *',
  daily: '0 2 * * *',
  weekly: '0 2 * * 0',
};

const PRESET_LABELS: Record<string, string> = {
  disabled: 'Disabled',
  hourly: 'Hourly',
  every6h: 'Every 6h',
  daily: 'Daily',
  weekly: 'Weekly',
  custom: 'Custom (one-shot)',
};

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

// Map a (date, time) pair to the canonical one-shot cron
// `min hr dom mon dow` where dow is `*`. Returns null when the
// inputs are empty or invalid.
export function cronFromDateTime(date: string, time: string): string | null {
  if (!date || !time) return null;
  // date is YYYY-MM-DD, time is HH:MM
  const dateMatch = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  const timeMatch = /^(\d{2}):(\d{2})$/.exec(time);
  if (!dateMatch || !timeMatch) return null;
  // Year is captured but not used (cron has no year field).
  const mo = Number(dateMatch[2]);
  const d = Number(dateMatch[3]);
  const h = Number(timeMatch[1]);
  const mi = Number(timeMatch[2]);
  // Sanity ranges (the native pickers constrain these but
  // a pasted value could be anything).
  if (mo < 1 || mo > 12 || d < 1 || d > 31 || h < 0 || h > 23 || mi < 0 || mi > 59) return null;
  return `${mi} ${h} ${d} ${mo} *`;
}

// Inverse of cronFromDateTime. Returns null when the cron is
// not a one-shot (i.e. any field is `*`, `/`, or `-`).
export function dateTimeFromCron(cron: string): { date: string; time: string } | null {
  const trimmed = cron.trim();
  const fields = trimmed.split(/\s+/);
  if (fields.length !== 5) return null;
  const [mi, hr, dom, mon, dow] = fields;
  // one-shot: every field is a single number, no wildcards, dow = *
  const isPlain = (s: string) => /^\d+$/.test(s);
  if (!isPlain(mi) || !isPlain(hr) || !isPlain(dom) || !isPlain(mon) || dow !== '*') return null;
  const today = new Date();
  const y = today.getFullYear();
  return {
    date: `${y}-${String(mon).padStart(2, '0')}-${String(dom).padStart(2, '0')}`,
    time: `${String(hr).padStart(2, '0')}:${String(mi).padStart(2, '0')}`,
  };
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
  const [validationError, setValidationError] = useState('');

  // ponytail: derive the active cron expression from either the
  // picker (when schedule === 'custom') or the preset map. The
  // picker's cron is computed; the parent's customSchedule is the
  // source of truth while editing.
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
    setValidationError(customSchedule && !validateCron(customSchedule) ? 'Invalid cron expression' : '');
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
      setValidationError('Pick a date and a time');
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
      setValidationError('Invalid cron expression');
    }
  };

  return (
    <div className="space-y-4">
      <label className="space-y-2 block">
        <span className="text-sm font-medium text-base-content">Schedule</span>
        <select
          data-testid="preset-select"
          value={schedule}
          onChange={handlePresetSelect}
          className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
        >
          {(['disabled', 'hourly', 'every6h', 'daily', 'weekly', 'custom'] as PresetType[]).map((key) => (
            <option key={key} value={key} data-testid={`preset-${key}`}>
              {PRESET_LABELS[key]}
            </option>
          ))}
        </select>
      </label>

      {schedule !== 'disabled' && (
        <div className="rounded-lg border border-border bg-base-content/5 px-3 py-2 text-sm text-base-content/75">
          {schedule !== 'custom' && activeCron && (
            <span>Cron: <code className="font-mono">{activeCron}</code></span>
          )}
          {schedule === 'custom' && activeCron && (
            <span>
              Runs once on <strong>{customDate}</strong> at <strong>{customTime}</strong>{' '}
              (cron: <code className="font-mono">{activeCron}</code>)
            </span>
          )}
        </div>
      )}

      {schedule === 'custom' && (
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <label className="space-y-1">
              <span className="text-xs font-medium text-base-content/75">Date</span>
              <input
                type="date"
                data-testid="custom-date"
                value={customDate}
                onChange={(e) => handlePickerChange({ date: e.target.value, time: customTime })}
                className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
              />
            </label>
            <label className="space-y-1">
              <span className="text-xs font-medium text-base-content/75">Time</span>
              <input
                type="time"
                data-testid="custom-time"
                value={customTime}
                onChange={(e) => handlePickerChange({ date: customDate, time: e.target.value })}
                className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
              />
            </label>
          </div>

          <label className="flex items-center gap-2 text-sm text-base-content/75">
            <input
              type="checkbox"
              data-testid="custom-advanced-toggle"
              checked={showRawCron}
              onChange={(e) => setShowRawCron(e.target.checked)}
              className="h-4 w-4"
            />
            Advanced (edit raw cron)
          </label>

          {showRawCron && (
            <label className="space-y-2 block">
              <span className="text-sm font-medium text-base-content">Cron expression</span>
              <input
                type="text"
                data-testid="custom-cron-input"
                value={rawCron}
                onChange={(e) => setRawCron(e.target.value)}
                onBlur={handleRawCronBlur}
                placeholder="0 2 * * * (5-field format)"
                className={`glass-panel w-full rounded-xl border px-3 py-2 text-base-content focus:outline-none focus:ring-2 ${
                  validationError ? 'border-error focus:ring-error/50' : 'border-border focus:ring-primary/50'
                }`}
              />
              <p className="text-xs text-base-content/55">
                5-field format: minute hour day-of-month month day-of-week
              </p>
            </label>
          )}

          {validationError && (
            <p className="text-xs text-error">{validationError}</p>
          )}
        </div>
      )}

      {/* Save button + status indicator */}
      {onSave && (
        <div className="flex items-center justify-between gap-3 pt-2">
          <div className="flex items-center gap-2">
            {saveState === 'saving' && (
              <div className="flex items-center gap-2 text-sm text-base-content/60">
                <div className="h-3 w-3 rounded-full bg-primary/60 animate-pulse" />
                {UI_COPY.saving}
              </div>
            )}
            {saveState === 'saved' && (
              <div className="flex items-center gap-2 text-sm text-success">
                <span className="inline-block h-3 w-3 rounded-full bg-success" />
                {UI_COPY.saved}
              </div>
            )}
            {saveState === 'error' && saveError && (
              <div className="flex items-center gap-2 text-sm text-error">
                <span className="inline-block h-3 w-3 rounded-full bg-error" />
                {saveError}
              </div>
            )}
          </div>
          <button
            onClick={onSave}
            disabled={saveState === 'saving'}
            className="action-btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
            data-testid="cron-save-button"
          >
            {saveState === 'saving' ? UI_COPY.saving : 'Save Schedule'}
          </button>
        </div>
      )}
    </div>
  );
}
