import { StatusChip } from '@/shared/components/ui';

interface EdgeModeBadgeProps {
  /** Whether edge mode is enabled. Undefined (older backend) is treated as off. */
  enabled?: boolean;
}

/**
 * Read-only badge that surfaces the EDGE_MODE deployment flag (ADR 0002).
 * Enabled renders prominently with the info palette; disabled stays
 * neutral so non-edge deployments — the default — are not visually noisy.
 *
 * Migrated to <StatusChip /> in issue #766 slice B so every status
 * surface across the app uses the same CVA variants and palette.
 */
export function EdgeModeBadge({ enabled }: EdgeModeBadgeProps) {
  const on = Boolean(enabled);
  const variant = on ? 'info' : 'neutral';

  return (
    <StatusChip
      variant={variant}
      size="md"
      data-testid="edge-mode-badge"
    >
      {`Edge mode: ${on ? 'enabled' : 'disabled'}`}
    </StatusChip>
  );
}
