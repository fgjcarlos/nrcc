import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui';
import { getVersionVariant } from './versionSemantics';

/**
 * CompatModeChip — issue #766 slice C
 *
 * Persistent header chip that tells the operator what configuration
 * mode NRCC detected: editable (full editing in-contract) or read-only
 * (legacy Node-RED 4, or unknown major). Reads from
 * ConfigurationCapabilities on the hostStatus payload.
 */

export interface CompatModeChipProps {
  /**
   * Configuration edit mode. When undefined the chip renders the
   * neutral "loading" state until the parent query resolves.
   */
  editable?: boolean;
  /** Raw mode string from the backend. */
  mode?: 'editable' | 'read-only';
  /** Detected runtime version, used to choose the icon for read-only. */
  runtimeVersion?: string;
  /** Optional explanation surfaced as the aria-label when read-only. */
  reason?: string;
}

export function CompatModeChip({ editable, mode, runtimeVersion, reason }: CompatModeChipProps) {
  const { t } = useT();

  let variant: 'success' | 'warning' | 'danger' | 'neutral';
  let label: string;
  let ariaLabel: string;

  if (editable === undefined) {
    variant = 'neutral';
    label = t('common:appChrome.compatModeUnknown');
    ariaLabel = t('common:appChrome.compatModeAria', { state: 'unknown' });
  } else if (editable || mode === 'editable') {
    variant = 'success';
    label = t('common:appChrome.compatModeEditable');
    ariaLabel = t('common:appChrome.compatModeAria', { state: 'editable' });
  } else {
    // read-only mode — colour follows the same major-axis contract as
    // the version chip so the two chips read consistently.
    const versionVariant = getVersionVariant(runtimeVersion);
    variant = versionVariant === 'danger' ? 'danger' : 'warning';
    label = t('common:appChrome.compatModeReadOnly');
    ariaLabel = reason
      ? t('common:appChrome.compatModeAriaWithReason', { state: 'read-only', reason })
      : t('common:appChrome.compatModeAria', { state: 'read-only' });
  }

  return (
    <StatusChip
      variant={variant}
      size="sm"
      ariaLabel={ariaLabel}
      data-testid="compat-mode-chip"
    >
      {label}
    </StatusChip>
  );
}
