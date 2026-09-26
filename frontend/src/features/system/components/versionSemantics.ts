/**
 * Node-RED version semantics — issue #766 slice C.
 *
 * Extracted from NodeRedVersionChip.tsx so the .tsx file only exports
 * React components, which keeps Vite's fast-refresh working.
 *
 * Colour contract (aligned with issue #765 compatibility policy):
 *   - 5.x   → success  (full editing in-contract)
 *   - 4.x   → warning  (legacy, read-only)
 *   - other → danger   (unknown / parse failure)
 *   - n/a   → neutral  (loading or older backend omits the field)
 */

import type { StatusChipVariantProps } from '@/shared/components/ui';

export type ChipVariant = NonNullable<StatusChipVariantProps['variant']>;

export function parseMajorVersion(version: string | null | undefined): number | null {
  if (!version) return null;
  const match = /^v?(\d+)/.exec(version.trim());
  return match ? Number(match[1]) : null;
}

export function getVersionVariant(version: string | null | undefined): ChipVariant {
  if (!version) return 'neutral';
  const major = parseMajorVersion(version);
  if (major === null) return 'danger';
  if (major >= 5) return 'success';
  if (major === 4) return 'warning';
  return 'danger';
}
