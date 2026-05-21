import { type ButtonHTMLAttributes, type Ref } from 'react';
import { type VariantProps } from 'class-variance-authority';
import { cn } from '@/shared/lib/utils';
import { buttonVariants } from '@/shared/constants/buttonVariants';

export interface ButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  ref?: Ref<HTMLButtonElement>;
}

export const Button = ({
  className,
  variant,
  size,
  ref,
  ...props
}: ButtonProps) => {
  return (
    <button
      ref={ref}
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  );
};
