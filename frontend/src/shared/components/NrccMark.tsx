import { type ReactNode, useId } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/shared/lib/utils';

/**
 * NrccMark: single source of truth for the NRCC brand mark (Header,
 * Sidebar, Login). The glyph is a drawing title block: a solid tile
 * split into a 2x2 grid that spells N R / C C. Letters and grid rules
 * are knocked out of the tile with a mask, so the ground shows through
 * them on any surface and the tile takes the tone's text colour.
 * Letters are stroke paths, not text, so the mark never depends on a
 * font. Drawn on a 32-unit grid; each cell is 16 units.
 */
const MARK_LETTERS = {
  N: 'M4.5 12.5V3.5L11.5 12.5V3.5',
  R: 'M20.5 12.5V3.5H24.75A2.25 2.25 0 0 1 24.75 8H20.5M24 8L27.5 12.5',
  C1: 'M11.2 20.8A4.5 4.5 0 1 0 11.2 27.2',
  C2: 'M27.2 20.8A4.5 4.5 0 1 0 27.2 27.2',
} as const;

function TitleBlock() {
  const maskId = `nrcc-mark-${useId().replace(/:/g, '')}`;
  return (
    <svg viewBox="0 0 32 32" aria-hidden="true" focusable="false">
      <mask id={maskId} maskUnits="userSpaceOnUse" x="0" y="0" width="32" height="32">
        <rect width="32" height="32" fill="white" />
        <g fill="none" stroke="black">
          <line x1="2" y1="16" x2="30" y2="16" strokeWidth="1.5" />
          <line x1="16" y1="2" x2="16" y2="30" strokeWidth="1.5" />
          <g strokeWidth="2.25" strokeLinejoin="miter">
            {Object.entries(MARK_LETTERS).map(([key, d]) => (
              <path key={key} d={d} data-letter={key} />
            ))}
          </g>
        </g>
      </mask>
      <rect width="32" height="32" fill="currentColor" mask={`url(#${maskId})`} />
    </svg>
  );
}

const nrccMarkVariants = cva(
  'inline-flex shrink-0 items-center justify-center ' +
    'transition-colors focus-visible:outline-none focus-visible:ring-2 ' +
    'focus-visible:ring-offset-2 focus-visible:ring-offset-base-100 ' +
    'focus-visible:ring-ds-accent-primary [&_svg]:h-full [&_svg]:w-full',
  {
    variants: {
      tone: {
        // Used by the Header topbar mark.
        accent: 'text-ds-accent-primary',
        primary: 'text-ds-brand-primary',
        // Subdued mark used inside dense rows.
        muted: 'text-base-content/70',
        // Used by the ink navigation rail (Sidebar): a light tile.
        rail: 'text-[var(--ds-rail-text)]',
      },
      size: {
        sm: 'h-9 w-9',
        md: 'h-11 w-11',
        lg: 'h-14 w-14',
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
  /** Optional override of the inner glyph (defaults to the title block). */
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
  const inner = icon ?? <TitleBlock />;

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
