import { useState, useEffect } from 'react';
import { ChevronDown } from 'lucide-react';

const PRESET_CRON: Record<string, string> = {
  hourly: '0 * * * *',
  every6h: '0 */6 * * *',
  daily: '0 2 * * *',
  weekly: '0 2 * * 0',
};

// Basic 5-field cron validation
// Simplified: allows numbers, *, /, -, comma separators
// Validates field ranges: min (0-59), hr (0-23), dom (1-31), mon (1-12), dow (0-6)
export function validateCron(expr: string): boolean {
  const trimmed = expr.trim();
  
  // Must be exactly 5 fields separated by spaces
  const fields = trimmed.split(/\s+/);
  if (fields.length !== 5) {
    return false;
  }

  // Basic check: each field should only contain numbers, *, /, -, or comma
  const validFieldRegex = /^[\d,*/-]+$/;
  if (!fields.every(field => validFieldRegex.test(field))) {
    return false;
  }

  // More strict: validate ranges for each field
  // minute: 0-59, hour: 0-23, day: 1-31, month: 1-12, dow: 0-6
  const [minute, hour, day, month, dow] = fields;
  const ranges = [
    { field: minute, min: 0, max: 59 },
    { field: hour, min: 0, max: 23 },
    { field: day, min: 1, max: 31 },
    { field: month, min: 1, max: 12 },
    { field: dow, min: 0, max: 6 },
  ];

  for (const { field, min, max } of ranges) {
    if (field === '*') continue;

    // Extract all numbers from the field
    const numbers = field.split(/[,/-]/).filter(n => n.length > 0 && n !== '*');
    
    for (const numStr of numbers) {
      const num = Number(numStr);
      if (!Number.isInteger(num) || num < min || num > max) {
        return false;
      }
    }
  }

  return true;
}

export type SaveState = 'idle' | 'saving' | 'saved' | 'error';
export type PresetType = 'hourly' | 'every6h' | 'daily' | 'weekly' | 'custom' | null;

export interface CronBuilderProps {
  value: string;
  onChange: (cron: string) => void;
  onPresetChange?: (preset: PresetType) => void;
  onSave?: () => void;
  saveState?: SaveState;
  saveError?: string;
}

export function CronBuilder({ value, onChange, onPresetChange, onSave, saveState = 'idle', saveError }: CronBuilderProps) {
  const [preset, setPreset] = useState<string | null>(null);
  const [customValue, setCustomValue] = useState('');
  const [validationError, setValidationError] = useState('');

  // Initialize state from props
  useEffect(() => {
    if (value === 'disabled' || value === '') {
      setPreset(null);
      setCustomValue('');
      setValidationError('');
    } else if (Object.values(PRESET_CRON).includes(value)) {
      // Find which preset this cron belongs to
      const presetKey = Object.entries(PRESET_CRON).find(([, cron]) => cron === value)?.[0];
      setPreset(presetKey || null);
      setCustomValue('');
      setValidationError('');
    } else {
      // Must be custom
      setPreset('custom');
      setCustomValue(value);
      if (value && !validateCron(value)) {
        setValidationError('Invalid cron expression');
      } else {
        setValidationError('');
      }
    }
  }, [value]);

  const handlePresetClick = (presetKey: string) => {
    setPreset(presetKey);
    setValidationError('');

    if (presetKey === 'custom') {
      setCustomValue('');
      // Don't call onChange for custom - wait for custom value input
      // Emit preset change to parent
      onPresetChange?.('custom');
    } else {
      setCustomValue('');
      // Emit the preset type to parent (not the cron string)
      onPresetChange?.(presetKey as PresetType);
      // Also emit the actual cron value
      onChange(PRESET_CRON[presetKey]);
    }
  };

  const handleCustomChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newValue = e.target.value;
    setCustomValue(newValue);
    setValidationError('');
  };

  const handleCustomBlur = () => {
    if (!customValue.trim()) {
      return;
    }

    const trimmed = customValue.trim();
    if (validateCron(trimmed)) {
      setValidationError('');
      // Emit the custom cron value
      onChange(trimmed);
      // Emit preset type as 'custom'
      onPresetChange?.('custom');
    } else {
      setValidationError('Invalid cron expression');
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-3">
        <label className="space-y-2">
          <span className="text-sm font-medium text-base-content">Scheduler Presets</span>
          <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
            {Object.keys(PRESET_CRON).map((key) => (
              <label key={key} className="flex items-center gap-2">
                <input
                  type="radio"
                  name="cron-preset"
                  value={key}
                  data-testid={`preset-${key}`}
                  checked={preset === key}
                  onChange={() => handlePresetClick(key)}
                  className="h-4 w-4"
                />
                <span className="capitalize text-sm text-base-content">
                  {key === 'every6h' ? 'Every 6h' : key}
                </span>
              </label>
            ))}
            <label className="flex items-center gap-2">
              <input
                type="radio"
                name="cron-preset"
                value="custom"
                data-testid="preset-custom"
                checked={preset === 'custom'}
                onChange={() => handlePresetClick('custom')}
                className="h-4 w-4"
              />
              <span className="text-sm text-base-content">Custom</span>
            </label>
          </div>
        </label>
      </div>

      {preset === 'custom' && (
        <label className="space-y-2">
          <span className="text-sm font-medium text-base-content">Custom Cron Expression</span>
          <input
            type="text"
            value={customValue}
            onChange={handleCustomChange}
            onBlur={handleCustomBlur}
            placeholder="0 2 * * * (format: min hr dom mon dow)"
            className={`glass-panel w-full rounded-xl border px-3 py-2 text-base-content focus:outline-none focus:ring-2 ${
              validationError
                ? 'border-error focus:ring-error/50'
                : 'border-border focus:ring-primary/50'
            }`}
          />
          <p className="text-xs text-base-content/55">5-field format: minute hour day-of-month month day-of-week</p>
          {validationError && (
            <p className="text-xs text-error">{validationError}</p>
          )}
        </label>
      )}

      {/* Save button + status indicator */}
      {onSave && (
        <div className="flex items-center justify-between gap-3 pt-2">
          <div className="flex items-center gap-2">
            {saveState === 'saving' && (
              <div className="flex items-center gap-2 text-sm text-base-content/60">
                <div className="h-3 w-3 rounded-full bg-primary/60 animate-pulse" />
                Guardando...
              </div>
            )}
            {saveState === 'saved' && (
              <div className="flex items-center gap-2 text-sm text-success">
                <span className="inline-block h-3 w-3 rounded-full bg-success" />
                Guardado
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
            {saveState === 'saving' ? 'Saving...' : 'Save Schedule'}
          </button>
        </div>
      )}
    </div>
  );
}
