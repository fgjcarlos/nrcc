import type { HTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';
import { useT } from '@/i18n';

/**
 * Skeleton — issue #766 slice G W1
 *
 * Loading placeholder that preserves layout while data is pending.
 * The outer div is the screen-reader-landmark (`role="status"` +
 * `aria-busy="true"` + a visually-hidden label) and the inner divs are
 * the visual blocks (`rows` of them), drawn on the sunken ground. The shimmer respects `prefers-reduced-motion`
 * via Tailwind's `motion-reduce:animate-none` utility so users who
 * have opted out of motion see a static block.
 */

export type SkeletonVariant = 'text' | 'rect' | 'circle';

export interface SkeletonProps extends HTMLAttributes<HTMLDivElement> {
  variant?: SkeletonVariant;
  /** Number of blocks inside the one status region. */
  rows?: number;
}

const variantClasses: Record<SkeletonVariant, string> = {
  text: 'h-3 w-full rounded-box',
  rect: 'h-32 w-full rounded-box',
  circle: 'h-12 w-12 rounded-full',
};

export function Skeleton({ variant = 'text', rows = 1, className, ...rest }: SkeletonProps) {
  const { t } = useT();

  return (
    <div
      role="status"
      aria-busy="true"
      aria-label={t('common:loading')}
      className="space-y-2"
      {...rest}
    >
      {Array.from({ length: rows }, (_, index) => (
        <div
          key={index}
          aria-hidden="true"
          className={cn(
            'bg-ds-bg-base motion-reduce:animate-none',
            'animate-pulse',
            variantClasses[variant],
            className,
          )}
        />
      ))}
    </div>
  );
}