import { type InputHTMLAttributes, type Ref } from 'react';
import { cn } from '@/shared/lib/utils';

export interface InputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> {
  error?: boolean;
  label?: string;
  ref?: Ref<HTMLInputElement>;
}

export const Input = ({
  className,
  error,
  label,
  id,
  ref,
  type,
  ...props
}: InputProps) => {
  const inputId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);

  return (
    <div className="space-y-1.5">
      {label && (
        <label
          htmlFor={inputId}
          className="text-label text-base-content"
        >
          {label}
        </label>
      )}
      <input
        ref={ref}
        id={inputId}
        // Disable password-manager autofill on password inputs by default:
        // sites that legitimately *change* a password should opt in to
        // autocomplete="current-password" or "new-password" explicitly. Without
        // this, browsers auto-fill the user's stored password into unrelated
        // /configuration fields and trigger the 400 in #706. Inputs that
        // really need an existing password (e.g. the login form) override
        // this via the explicit `autoComplete` prop.
        autoComplete={type === 'password' ? 'off' : undefined}
        type={type}
        className={cn(
          'flex w-full rounded-md px-3 py-2 text-sm',
          'bg-base-300 text-base-content placeholder:text-base-content/50',
          'border-none outline-none',
          'transition-all duration-200',
          'file:border-0 file:bg-transparent file:text-sm file:font-medium',
          'focus:ring-1 focus:ring-primary input-focus-glow',
          error && 'ring-1 ring-error input-error-glow',
          'disabled:cursor-not-allowed disabled:opacity-50',
          className
        )}
        {...props}
      />
    </div>
  );
};
