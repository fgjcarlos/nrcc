import { type ReactNode, type TableHTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

export interface TableProps extends TableHTMLAttributes<HTMLTableElement> {
  /** Accessible name, rendered as a visually hidden caption. */
  caption?: string;
  /**
   * Below `sm`, render each row as a labelled card: every `<td>` shows its
   * `data-label` beside the value, so no column scrolls off a phone screen.
   */
  stacked?: boolean;
  children: ReactNode;
}

/**
 * Shared data table (DESIGN.md: hairline row rules, 12px sentence-case
 * headers on the sunken ground, 14px cells, tabular numerals, dense rows).
 * Styling lives on `.ds-table` in src/index.css; write plain
 * thead/tbody/tr/th/td inside, with `data-label` on each td when stacked.
 */
export function Table({ caption, stacked = false, className, children, ...props }: TableProps) {
  return (
    <div className="ds-table-wrap" data-stacked={stacked || undefined}>
      <table className={cn('ds-table', className)} {...props}>
        {caption && <caption className="sr-only">{caption}</caption>}
        {children}
      </table>
    </div>
  );
}
