import { type Ref, type SelectHTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';
import { FieldShell } from './FieldShell';
import { fieldControlClass, useField } from './useField';

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  /** `true` marks the field invalid; a string is also rendered as its message. */
  error?: boolean | string;
  label?: string;
  ref?: Ref<HTMLSelectElement>;
}

/** Native select on the shared control boundary (extracted from FormFields' SelectField). */
export const Select = ({
  className,
  error,
  label,
  id,
  ref,
  'aria-describedby': describedBy,
  'aria-invalid': ariaInvalid,
  ...props
}: SelectProps) => {
  const field = useField({ id, label, error, 'aria-describedby': describedBy, 'aria-invalid': ariaInvalid });

  return (
    <FieldShell field={field} label={label}>
      <select ref={ref} className={cn(fieldControlClass, className)} {...props} {...field.controlProps} />
    </FieldShell>
  );
};
