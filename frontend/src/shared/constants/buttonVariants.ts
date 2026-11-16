import { cva, type VariantProps } from 'class-variance-authority';

/**
 * Button variants: one button system. Each variant is an `.action-btn-*`
 * class from src/index.css (DESIGN.md Buttons), the same classes the raw
 * call sites use; daisyUI `.btn` is restyled to the same values there.
 * md is the class default (36px); sm and lg only change the target.
 * Extracted from Button.tsx to resolve fast-refresh lint warning.
 */
export const buttonVariants = cva('', {
  variants: {
    variant: {
      // Action: sheet ground, conductor-blue edge and label.
      primary: 'action-btn-primary',
      // Conductor-blue fill.
      solid: 'action-btn-solid',
      secondary: 'action-btn-secondary',
      ghost: 'action-btn-ghost',
      // Destructive actions only.
      danger: 'action-btn-danger',
    },
    size: {
      sm: 'min-h-8 px-3 py-1 text-xs',
      md: '',
      lg: 'min-h-11 px-6',
    },
  },
  defaultVariants: {
    variant: 'primary',
    size: 'md',
  },
});

export type ButtonVariantProps = VariantProps<typeof buttonVariants>;
