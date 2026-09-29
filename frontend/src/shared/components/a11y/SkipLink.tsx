import { useT } from '@/i18n';

/**
 * SkipLink — issue #766 slice G W2
 *
 * Visually-hidden anchor that becomes visible on focus and jumps the
 * keyboard user past the chrome to the page's main landmark. Mount it
 * as the very first focusable element in the layout so the very first
 * Tab press reveals it.
 *
 * The target element (typically `<main id="main-content" tabindex="-1">`)
 * must opt in to receive programmatic focus. We pass `tabIndex={-1}`
 * to the consumer's main wrapper (see Layout.tsx) so the browser
 * scrolls and focuses it when the user activates this link.
 */

export interface SkipLinkProps {
  /** DOM id of the element to jump to. */
  targetId: string;
  /** Accessible name shown while focused. i18n-ready. */
  label: string;
}

export function SkipLink({ targetId, label }: SkipLinkProps) {
  // useT is imported so consumers can pass an already-translated label
  // (the call site does the lookup); this keeps the component
  // presenter-pure and the test surface deterministic.
  useT();
  return (
    <a
      href={`#${targetId}`}
      className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[200] focus:rounded-lg focus:bg-base-100 focus:px-4 focus:py-2 focus:text-sm focus:font-semibold focus:text-base-content focus:shadow-glow focus:outline-none focus:ring-2 focus:ring-ds-accent-primary focus:ring-offset-2 focus:ring-offset-base-100"
    >
      {label}
    </a>
  );
}