import { type ReactNode } from 'react';
import { cn } from '@/shared/lib/utils';
import { sectionId } from './lib/tokens';

/** A styleguide section: a region named by its heading, ranked by a control-border rule. */
export function Section({ title, intro, children }: { title: string; intro?: ReactNode; children: ReactNode }) {
  const id = sectionId(title);
  return (
    <section id={id} aria-labelledby={`${id}-title`} className="scroll-mt-20 border-t border-ds-border-strong pt-4">
      <h2 id={`${id}-title`} className="text-sm font-bold text-ds-text-primary">
        {title}
      </h2>
      {intro && <p className="mt-1 max-w-prose text-sm text-ds-text-secondary">{intro}</p>}
      <div className="mt-4 space-y-6">{children}</div>
    </section>
  );
}

/** One specimen: name, when to use it (from DESIGN.md), then the real component. */
export function Specimen({
  name,
  caption,
  children,
  className,
}: {
  name: string;
  caption: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className="space-y-2">
      <h3 className="text-sm font-semibold text-ds-text-primary">{name}</h3>
      <p className="max-w-prose text-sm text-ds-text-secondary">{caption}</p>
      <div className={cn('rounded-box border border-ds-border-default bg-ds-bg-surface p-4', className)}>{children}</div>
    </div>
  );
}

/** Small state or variant label beside a specimen. */
export function Tag({ children }: { children: ReactNode }) {
  return <code className="font-mono text-xs font-semibold text-ds-text-muted">{children}</code>;
}
