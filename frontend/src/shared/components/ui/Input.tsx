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
