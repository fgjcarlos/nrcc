import { type HTMLAttributes, type ReactNode } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/shared/lib/utils';

/**
 * StatusChip — issue #766 slice B
 *
 * Reusable status surface used by Configuration, Security, Environment,
 * Recovery and Overview to communicate the state of an item at a glance.
 * Replaces the inline chip pattern (rounded-full, padding-3, small text,
 * 15-percent accent background) currently duplicated across pages
 * (e.g. EdgeModeBadge).
 *
 * Variants map to the existing ds-* semantic palette in
 * tailwind.config.js — no new colours are introduced.
 */

export const statusChipVariants = cva(
  'inline-flex items-center gap-1.5 rounded-full font-medium transition-colors ' +
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-1 ' +
    'focus-visible:ring-offset-base-100 focus-visible:ring-ds-accent-primary',
  {
    variants: {
      variant: {
        success: 'bg-ds-success/15 text-ds-success border border-ds-success/35',
        warning: 'bg-ds-warning/15 text-ds-warning border border-ds-warning/35',
        danger: 'bg-ds-danger/15 text-ds-danger border border-ds-danger/35',
        info: 'bg-ds-info/15 text-ds-info border border-ds-info/35',
        neutral: 'bg-base-300/60 text-base-content/80 border border-ds-border-default',
      },
      size: {
        // Only padding + font-size here. The variant owns the text
        // colour so tailwind-merge cannot resolve a conflict.
        sm: 'px-2.5 py-0.5 text-xs',
        md: 'px-3 py-1 text-sm',
      },
    },
    defaultVariants: {
      variant: 'neutral',
      size: 'md',
    },
  },
);

export type StatusChipVariantProps = VariantProps<typeof statusChipVariants>;

export interface StatusChipProps
  extends Omit<StatusChipVariantProps, 'variant' | 'size'>,
    Omit<HTMLAttributes<HTMLSpanElement>, 'children'> {
  variant?: StatusChipVariantProps['variant'];
  size?: StatusChipVariantProps['size'];
  /** Optional icon (lucide-react) rendered to the left of the label. */
  icon?: ReactNode;
  /** Visible label. Required for accessibility. */
  children: ReactNode;
  /** Optional accessible description; defaults to the visible label. */
  ariaLabel?: string;
}

/**
 * @example
 *   <StatusChip variant="success" size="sm">Saved</StatusChip>
 *   <StatusChip variant="warning" icon={<AlertTriangle className="h-3.5 w-3.5" />}>Restart required</StatusChip>
 */
export function StatusChip({
  variant,
  size,
  icon,
  children,
  ariaLabel,
  className,
  ...rest
}: StatusChipProps) {
  return (
    <span
      role="status"
      aria-label={ariaLabel ?? (typeof children === 'string' ? children : undefined)}
      className={cn(statusChipVariants({ variant, size }), className)}
      {...rest}
    >
      {icon ? <span aria-hidden="true">{icon}</span> : null}
      <span>{children}</span>
    </span>
  );
}
