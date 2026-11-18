import { useRef, type ElementType, type HTMLAttributes, type ReactNode } from 'react';
import { cn } from '@/shared/lib/utils';

export interface TabItem<T extends string = string> {
  id: T;
  label: ReactNode;
  icon?: ElementType;
}

export interface TabsProps<T extends string = string> {
  /** Accessible name of the tablist. */
  label: string;
  /** Prefix of the tab and panel ids: `${idBase}-tab-${id}`, `${idBase}-panel-${id}`. */
  idBase: string;
  items: TabItem<T>[];
  selected: T;
  onSelect: (id: T) => void;
  className?: string;
}

/** Roving tabindex: the next tab index for Left/Right/Home/End (wrapping), else null. */
function nextTabIndex(key: string, index: number, count: number): number | null {
  if (key === 'ArrowRight') return (index + 1) % count;
  if (key === 'ArrowLeft') return (index - 1 + count) % count;
  if (key === 'Home') return 0;
  if (key === 'End') return count - 1;
  return null;
}

/**
 * WAI-ARIA tablist (DESIGN.md Navigation › Tabs): roving tabindex, automatic
 * activation on Left/Right/Home/End, 36px tabs on a hairline baseline, the
 * selected tab in conductor blue over a 2px underline. Pair with TabPanel.
 */
export function Tabs<T extends string>({ label, idBase, items, selected, onSelect, className }: TabsProps<T>) {
  const tabRefs = useRef<Array<HTMLButtonElement | null>>([]);

  return (
    <div
      role="tablist"
      aria-label={label}
      className={cn('flex gap-1 overflow-x-auto border-b border-ds-border-subtle', className)}
    >
      {items.map((item, index) => {
        const Icon = item.icon;
        const isSelected = item.id === selected;
        return (
          <button
            key={item.id}
            ref={(el) => { tabRefs.current[index] = el; }}
            type="button"
            role="tab"
            id={`${idBase}-tab-${item.id}`}
            aria-selected={isSelected}
            aria-controls={`${idBase}-panel-${item.id}`}
            tabIndex={isSelected ? 0 : -1}
            onClick={() => onSelect(item.id)}
            onKeyDown={(event) => {
              const next = nextTabIndex(event.key, index, items.length);
              if (next === null) return;
              event.preventDefault();
              onSelect(items[next].id);
              tabRefs.current[next]?.focus();
            }}
            className={cn(
              '-mb-px flex min-h-9 items-center gap-2 border-b-2 px-3 py-2 text-sm font-medium whitespace-nowrap transition-colors',
              isSelected
                ? 'border-primary text-primary'
                : 'border-transparent text-ds-text-muted hover:border-[var(--ds-border-strong)] hover:text-ds-text-primary'
            )}
          >
            {Icon && <Icon className="h-4 w-4" aria-hidden="true" />}
            {item.label}
          </button>
        );
      })}
    </div>
  );
}

export interface TabPanelProps extends HTMLAttributes<HTMLDivElement> {
  idBase: string;
  selected: string;
}

/** The panel of the selected tab, labelled by it. */
export function TabPanel({ idBase, selected, className, ...props }: TabPanelProps) {
  return (
    <div
      role="tabpanel"
      id={`${idBase}-panel-${selected}`}
      aria-labelledby={`${idBase}-tab-${selected}`}
      tabIndex={0}
      className={cn('focus-visible:outline-offset-4', className)}
      {...props}
    />
  );
}
