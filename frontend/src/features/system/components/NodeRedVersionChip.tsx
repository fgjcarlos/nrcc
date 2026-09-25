import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui';
import { getVersionVariant } from './versionSemantics';

/**
 * NodeRedVersionChip — issue #766 slice C
 *
 * Persistent header chip that tells the operator which Node-RED version
 * NRCC detected in the deployment. Reads the version from the
 * /api/system/info response (already on SystemInfo as
 * `nodeRedVersion?: string`).
 *
 * Colour semantics live in versionSemantics.ts so this .tsx file
 * only exports React components (keeps Vite fast-refresh happy).
 */

export interface NodeRedVersionChipProps {
  /** Raw version string from /api/system/info (for example "5.0.7"). */
  version?: string | null;
}

export function NodeRedVersionChip({ version }: NodeRedVersionChipProps) {
  const { t } = useT();
  const variant = getVersionVariant(version);

  const displayVersion = version ?? t('common:appChrome.nodeRedVersionUnknown');

  return (
    <StatusChip
      variant={variant}
      size="sm"
      ariaLabel={t('common:appChrome.nodeRedVersionAria', { version: displayVersion })}
      data-testid="node-red-version-chip"
    >
      {t('common:appChrome.nodeRedVersionLabel')} {displayVersion}
    </StatusChip>
  );
}
