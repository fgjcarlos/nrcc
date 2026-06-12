import { cn } from '@/shared/lib';

interface EdgeModeBadgeProps {
  /** Whether edge mode is enabled. Undefined (older backend) is treated as off. */
  enabled?: boolean;
}

/**
 * Read-only badge that surfaces the EDGE_MODE deployment flag (ADR 0002).
 * Enabled renders prominently; disabled stays muted so non-edge deployments —
 * the default — are not visually noisy.
 */
export function EdgeModeBadge({ enabled }: EdgeModeBadgeProps) {
  const on = Boolean(enabled);

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium',
        on ? 'bg-info/15 text-info' : 'bg-base-300/60 text-base-content/70'
      )}
      data-testid="edge-mode-badge"
    >
      {`Edge mode: ${on ? 'enabled' : 'disabled'}`}
    </span>
  );
}
