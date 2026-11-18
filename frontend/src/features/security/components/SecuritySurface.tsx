import { useEffect, useRef, useState, type ReactNode } from 'react';
import { ChevronDown } from 'lucide-react';
import { useT } from '@/i18n';

interface SecuritySurfaceProps {
  /** Stable anchor; `/security#<anchor>` expands and scrolls to the surface. */
  anchor: string;
  testId: string;
  title: string;
  status: ReactNode;
  /** One line saying what the surface protects. */
  summary: string;
  /** The surface form, behind an Edit disclosure. */
  children?: ReactNode;
  /** Trailing action for surfaces without a form here. */
  action?: ReactNode;
  /** Extra summary content (counts) under the line. */
  details?: ReactNode;
}

/**
 * One access boundary on /security drawn as a flat row: name, state chip,
 * what it protects, then its form behind a native <details>. The form stays
 * mounted while collapsed, so typed values survive collapse and re-expand.
 */
export function SecuritySurface({ anchor, testId, title, status, summary, children, action, details }: SecuritySurfaceProps) {
  const { t } = useT();
  const ref = useRef<HTMLElement>(null);
  const [open, setOpen] = useState(false);
  const [revealed, setRevealed] = useState(0);
  const titleId = `${testId}-title`;

  useEffect(() => {
    // Read window.location so the hash works on mount and on in-page changes.
    const reveal = () => {
      if (window.location.hash !== `#${anchor}`) return;
      setOpen(true);
      setRevealed((count) => count + 1);
    };
    reveal();
    window.addEventListener('hashchange', reveal);
    return () => window.removeEventListener('hashchange', reveal);
  }, [anchor]);

  // Scroll after the form has opened, so a short collapsed page can still scroll.
  useEffect(() => {
    if (revealed > 0) ref.current?.scrollIntoView?.({ block: 'start' });
  }, [revealed]);

  return (
    <section
      ref={ref}
      id={anchor}
      data-testid={testId}
      aria-labelledby={titleId}
      className="relative scroll-mt-32 border-b sm:scroll-mt-20 border-ds-border-subtle py-4"
    >
      <div className="min-w-0 space-y-1 sm:pr-36">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <h2 id={titleId} className="text-sm font-semibold text-ds-text-primary">
            {title}
          </h2>
          {status}
        </div>
        <p className="text-xs text-ds-text-secondary">{summary}</p>
        {details}
      </div>
      {action && <div className="mt-2 sm:absolute sm:right-0 sm:top-3 sm:mt-0">{action}</div>}
      {children && (
        <details open={open} onToggle={(event) => setOpen(event.currentTarget.open)} className="group">
          <summary
            className="mt-2 inline-flex min-h-11 cursor-pointer list-none items-center gap-1.5 rounded-field border border-ds-border-strong bg-ds-bg-surface px-4 text-sm font-medium text-ds-text-primary hover:bg-base-content/5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ds-accent-primary sm:absolute sm:right-0 sm:top-3 sm:mt-0 md:min-h-9 [&::-webkit-details-marker]:hidden"
          >
            {t('security:surface.edit')}
            <span className="sr-only"> {title}</span>
            <ChevronDown className="h-4 w-4 transition-transform group-open:rotate-180 motion-reduce:transition-none" aria-hidden="true" />
          </summary>
          <div className="pt-4">{children}</div>
        </details>
      )}
    </section>
  );
}
