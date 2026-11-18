import { type InputHTMLAttributes, type Ref } from 'react';
import { cn } from '@/shared/lib/utils';
import { FieldShell } from './FieldShell';
import { fieldControlClass, useField } from './useField';

export interface InputProps
  extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> {
  /** `true` marks the field invalid; a string is also rendered as its message. */
  error?: boolean | string;
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
  'aria-describedby': describedBy,
  'aria-invalid': ariaInvalid,
  ...props
}: InputProps) => {
  const field = useField({ id, label, error, 'aria-describedby': describedBy, 'aria-invalid': ariaInvalid });

  return (
    <FieldShell field={field} label={label}>
      <input
        ref={ref}
        // Disable password-manager autofill on password inputs by default:
        // sites that legitimately *change* a password should opt in to
        // autocomplete="current-password" or "new-password" explicitly. Without
        // this, browsers auto-fill the user's stored password into unrelated
        // /configuration fields and trigger the 400 in #706. Inputs that
        // really need an existing password (e.g. the login form) override
        // this via the explicit `autoComplete` prop.
        autoComplete={type === 'password' ? 'off' : undefined}
        type={type}
        className={cn(fieldControlClass, 'file:border-0 file:bg-transparent file:text-sm file:font-medium', className)}
        {...props}
        {...field.controlProps}
      />
    </FieldShell>
  );
};
