import { Fragment } from 'react';
import type { LoggingLevel, NodeRedConfigFormData } from '@/shared/types';
import { Input } from '@/shared/components/ui';
import { SETTINGS_KEYS } from '../lib/fieldLabels';

type FieldKey = keyof NodeRedConfigFormData;

/**
 * Terminal label (T9 finish review): the settings.js key in mono as the
 * field's designator, then the human label. The control's accessible
 * name reads "UI Port (uiPort)" so speech and screen-reader users get the
 * plain label first.
 */
function fieldIdentity(field: FieldKey | undefined, label: string) {
  const key = field ? SETTINGS_KEYS[field] : undefined;
  return {
    key,
    id: field ? `cfg-${field}` : `cfg-${label.toLowerCase().replace(/\W+/g, '-')}`,
    accessibleName: key ? `${label} (${key})` : label,
  };
}

/**
 * A settings.js key in mono that may wrap only after a dot
 * (`editorTheme.codeEditor.` / `options.theme`), never mid-word. `<wbr>`
 * adds no text, so the accessible name is unchanged.
 */
export function SettingKey({ path, className }: { path: string; className?: string }) {
  const parts = path.split('.');
  return (
    <span className={className ?? 'font-mono font-semibold text-ds-text-primary'}>
      {/* Fragments keep every part a direct text node of this span. */}
      {parts.map((part, i) => (
        <Fragment key={i}>
          {part}
          {i < parts.length - 1 && <>.<wbr /></>}
        </Fragment>
      ))}
    </span>
  );
}

export function FieldLabel({ label, settingKey, htmlFor }: { label: string; settingKey?: string; htmlFor?: string }) {
  return (
    <label htmlFor={htmlFor} className="flex flex-wrap items-baseline gap-x-1.5 text-xs text-ds-text-secondary">
      {settingKey && <SettingKey path={settingKey} />}
      {settingKey && <span aria-hidden="true">·</span>}
      <span>{label}</span>
    </label>
  );
}

interface InputFieldProps {
  label: string;
  /** Form field; its settings.js key becomes the terminal label. */
  field?: FieldKey;
  value: string | number;
  onChange: (value: string | number) => void;
  type?: 'text' | 'number' | 'password';
  placeholder?: string;
  help?: string;
  disabled?: boolean;
  error?: boolean;
}

export function InputField({ label, field, value, onChange, type = 'text', placeholder, help, disabled, error }: InputFieldProps) {
  const { key, id, accessibleName } = fieldIdentity(field, label);
  return (
    <div className="space-y-1">
      <FieldLabel label={label} settingKey={key} htmlFor={id} />
      <Input
        id={id}
        aria-label={accessibleName}
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
      {help && <p className="text-xs text-ds-text-muted">{help}</p>}
    </div>
  );
}

interface ToggleFieldProps {
  label: string;
  field?: FieldKey;
  value: boolean;
  onChange: (value: boolean) => void;
  help?: string;
  disabled?: boolean;
}

export function ToggleField({ label, field, value, onChange, help, disabled }: ToggleFieldProps) {
  const { key, id, accessibleName } = fieldIdentity(field, label);
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="min-w-0">
        <FieldLabel label={label} settingKey={key} htmlFor={id} />
        {help && <p className="text-xs text-ds-text-muted">{help}</p>}
      </div>
      <button
        id={id}
        type="button"
        onClick={() => onChange(!value)}
        disabled={disabled}
        aria-pressed={value}
        aria-label={accessibleName}
        // Off: control-border outline and a muted knob, so the track stays
        // visible on the surface in both themes (WCAG 1.4.11).
        className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition-colors ${
          value ? 'border-primary bg-primary' : 'border-[var(--ds-border-strong)] bg-ds-bg-base'
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
  field?: FieldKey;
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string }[];
  disabled?: boolean;
}

export function SelectField({ label, field, value, onChange, options, disabled }: SelectFieldProps) {
  const { key, id, accessibleName } = fieldIdentity(field, label);
  return (
    <div className="space-y-1">
      <FieldLabel label={label} settingKey={key} htmlFor={id} />
      <select
        id={id}
        aria-label={accessibleName}
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
