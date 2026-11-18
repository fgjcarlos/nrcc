import { type ReactNode } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/shared/lib/utils';

/**
 * NrccMark: single source of truth for the NRCC brand mark (Header,
 * Sidebar). The glyph is the IEC normally-open contact, closed: the
 * conductor runs in from each edge and ends in an open terminal ring,
 * and the blade lies flat on the line between the rings, drawn heavier
 * than the conductor so it reads as a part, not as wire. Every piece
 * ends under a ring's stroke, so the rings stay open on any ground.
 * Drawn on a 16-unit grid; the glyph sizes below are 1x, 1.5x and 2x.
 */
function ClosedContact() {
  return (
    <svg viewBox="0 0 16 16" fill="none" aria-hidden="true" focusable="false">
      <g stroke="currentColor">
        <line x1="0" y1="8" x2="2.25" y2="8" strokeWidth="1.5" />
        <line x1="13.75" y1="8" x2="16" y2="8" strokeWidth="1.5" />
        <line x1="5.75" y1="8" x2="10.25" y2="8" strokeWidth="2.5" />
        <circle cx="4" cy="8" r="1.75" strokeWidth="1.5" />
        <circle cx="12" cy="8" r="1.75" strokeWidth="1.5" />
      </g>
    </svg>
  );
}

const nrccMarkVariants = cva(
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
        md: 'h-11 w-11 [&_svg]:h-6 [&_svg]:w-6',
        lg: 'h-14 w-14 [&_svg]:h-8 [&_svg]:w-8',
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
  /** Optional override of the inner glyph (defaults to the closed contact). */
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
  const inner = icon ?? <ClosedContact />;

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
