import { type ReactNode } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { RadioTower } from 'lucide-react';
import { cn } from '@/shared/lib/utils';

/**
 * NrccMark — issue #766 slice B
 *
 * Single source of truth for the NRCC brand mark. Replaces the
 * duplicated mark containers that lived in:
 *   - Header.tsx      (topbar-signal-icon, h-11 w-11, accent colour)
 *   - Sidebar.tsx     (sidebar-brand-mark, h-11 w-11 / h-10 w-10,
 *                      primary colour)
 *   - LandingView     (decorative logo on the public landing page)
 *
 * The visual treatment mirrors the existing palette: rounded-square
 * container, 1px border, brand-red primary or accent-cyan variant,
 * lucide RadioTower icon (the signal-tower glyph already chosen in
 * Header.tsx).
 */

export const nrccMarkVariants = cva(
  'inline-flex shrink-0 items-center justify-center rounded-xl border ' +
    'transition-colors focus-visible:outline-none focus-visible:ring-2 ' +
    'focus-visible:ring-offset-2 focus-visible:ring-offset-base-100 ' +
    'focus-visible:ring-ds-accent-primary',
  {
    variants: {
      tone: {
        // Used by the Header topbar mark.
        accent: 'border-ds-border-default text-ds-accent-primary bg-ds-bg-elevated/60',
        // Used by the Sidebar brand mark.
        primary: 'border-ds-border-default text-ds-brand-primary bg-ds-bg-elevated/60',
        // Subdued mark used inside dense rows.
        muted: 'border-ds-border-default text-base-content/70 bg-base-300/60',
      },
      size: {
        sm: 'h-9 w-9 [&_svg]:h-4 [&_svg]:w-4',
        md: 'h-11 w-11 [&_svg]:h-5 [&_svg]:w-5',
        lg: 'h-14 w-14 [&_svg]:h-7 [&_svg]:w-7',
      },
    },
    defaultVariants: {
      tone: 'accent',
      size: 'md',
    },
  },
);

export type NrccMarkVariantProps = VariantProps<typeof nrccMarkVariants>;

export interface NrccMarkProps
  extends Omit<NrccMarkVariantProps, 'tone' | 'size'> {
  tone?: NrccMarkVariantProps['tone'];
  size?: NrccMarkVariantProps['size'];
  /** Accessible label for the mark. Defaults to "NRCC". */
  ariaLabel?: string;
  /** Optional click handler; renders a button when provided. */
  onClick?: () => void;
  /** Extra className for layout positioning (margins, etc.). */
  className?: string;
  /** Optional override of the inner icon (defaults to RadioTower). */
  icon?: ReactNode;
}

/**
 * @example
 *   <NrccMark tone="accent" size="md" />
 *   <NrccMark tone="primary" size="sm" onClick={...} />
 */
export function NrccMark({
  tone,
  size,
  ariaLabel = 'NRCC',
  onClick,
  className,
  icon,
}: NrccMarkProps) {
  const inner = icon ?? (
    <RadioTower className="stroke-[1.8]" aria-hidden="true" />
  );

  const classes = cn(nrccMarkVariants({ tone, size }), className);

  if (onClick) {
    return (
      <button
        type="button"
        onClick={onClick}
        aria-label={ariaLabel}
        className={classes}
        data-testid="nrcc-mark"
      >
        {inner}
      </button>
    );
  }

  return (
    <span
      role="img"
      aria-label={ariaLabel}
      className={classes}
      data-testid="nrcc-mark"
    >
      {inner}
    </span>
  );
}
