import { type Ref, type TextareaHTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';
import { FieldShell } from './FieldShell';
import { fieldControlClass, useField } from './useField';

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  /** `true` marks the field invalid; a string is also rendered as its message. */
  error?: boolean | string;
  label?: string;
  ref?: Ref<HTMLTextAreaElement>;
}

export const Textarea = ({
  className,
  error,
  label,
  id,
  ref,
  'aria-describedby': describedBy,
  'aria-invalid': ariaInvalid,
  ...props
}: TextareaProps) => {
  const field = useField({ id, label, error, 'aria-describedby': describedBy, 'aria-invalid': ariaInvalid });

  return (
    <FieldShell field={field} label={label}>
      <textarea ref={ref} className={cn(fieldControlClass, 'resize-y', className)} {...props} {...field.controlProps} />
    </FieldShell>
  );
};
