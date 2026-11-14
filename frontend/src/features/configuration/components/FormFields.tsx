import type { LoggingLevel } from '@/shared/types';
import { Input } from '@/shared/components/ui';

interface InputFieldProps {
  label: string;
  value: string | number;
  onChange: (value: string | number) => void;
  type?: 'text' | 'number' | 'password';
  placeholder?: string;
  help?: string;
  disabled?: boolean;
  error?: boolean;
}

export function InputField({ label, value, onChange, type = 'text', placeholder, help, disabled, error }: InputFieldProps) {
  return (
    <div className="space-y-1">
      <Input
        label={label}
        type={type}
        value={value}
        onChange={(e) => onChange(type === 'number' ? Number(e.target.value) : e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        error={error}
        // The shared Input sets border-none; the global input rule supplies
        // the control-border colour (--ds-border-strong, >=3:1, WCAG 1.4.11).
        className="border border-solid"
      />
      {help && <p className="text-xs text-base-content/60">{help}</p>}
    </div>
  );
}

interface ToggleFieldProps {
  label: string;
  value: boolean;
  onChange: (value: boolean) => void;
  help?: string;
  disabled?: boolean;
}

export function ToggleField({ label, value, onChange, help, disabled }: ToggleFieldProps) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <label className="text-sm font-medium text-base-content">{label}</label>
        {help && <p className="text-xs text-base-content/60">{help}</p>}
      </div>
      <button
        type="button"
        onClick={() => onChange(!value)}
        disabled={disabled}
        aria-pressed={value}
        // Off: control-border outline and a muted knob, so the track stays
        // visible on the surface in both themes (WCAG 1.4.11).
        className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition-colors ${
          value ? 'border-primary bg-primary' : 'border-ds-border-default bg-ds-bg-base'
        } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
      >
        <span
          className={`inline-block h-4 w-4 transform rounded-full transition-transform ${
            value ? 'translate-x-6 bg-primary-content' : 'translate-x-1 bg-ds-text-muted'
          }`}
        />
      </button>
    </div>
  );
}

interface SelectFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
  disabled?: boolean;
}

export function SelectField({ label, value, onChange, options, disabled }: SelectFieldProps) {
  return (
    <div className="space-y-1">
      <label className="text-label text-base-content">{label}</label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        className="flex w-full rounded-md px-3 py-2 text-sm bg-base-300 text-base-content border border-solid outline-none transition-all duration-200 focus:ring-1 focus:ring-primary input-focus-glow disabled:cursor-not-allowed disabled:opacity-50"
      >
        {options.map((opt) => (
          <option key={opt.value} value={opt.value}>{opt.label}</option>
        ))}
      </select>
    </div>
  );
}

const LOGGING_LEVELS: { value: LoggingLevel; label: string }[] = [
  { value: 'trace', label: 'Trace' },
  { value: 'debug', label: 'Debug' },
  { value: 'info', label: 'Info' },
  { value: 'warn', label: 'Warning' },
  { value: 'error', label: 'Error' },
  { value: 'fatal', label: 'Fatal' },
  { value: 'off', label: 'Off' },
];

export { LOGGING_LEVELS };
