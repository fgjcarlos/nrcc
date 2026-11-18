import { type ButtonHTMLAttributes, type Ref } from 'react';
import { cn } from '@/shared/lib/utils';

export interface ToggleProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onChange' | 'value' | 'type'> {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  ref?: Ref<HTMLButtonElement>;
}

/**
 * DESIGN.md toggle, extracted from FormFields' ToggleField: a 44x24px round
 * track exposing aria-pressed. Off is a control-border outline on the
 * sunken ground with a muted knob, so the track stays visible on the
 * surface in both themes (WCAG 1.4.11); on is a conductor-blue fill.
 * The caller supplies the accessible name (aria-label or a <label>).
 */
export function Toggle({ checked, onCheckedChange, className, ref, ...props }: ToggleProps) {
  return (
    <button
      ref={ref}
      {...props}
      type="button"
      aria-pressed={checked}
      onClick={() => onCheckedChange(!checked)}
      className={cn(
        'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition-colors',
        'disabled:cursor-not-allowed disabled:opacity-50',
        checked ? 'border-primary bg-primary' : 'border-ds-border-strong bg-ds-bg-base',
        className,
      )}
    >
      <span
        aria-hidden="true"
        className={cn(
          'inline-block h-4 w-4 rounded-full transition-transform',
          checked ? 'translate-x-6 bg-primary-content' : 'translate-x-1 bg-ds-text-muted',
        )}
      />
    </button>
  );
}
