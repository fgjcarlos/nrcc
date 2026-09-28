/**
 * FieldStatusChip — issue #766 slice F W2.
 *
 * Inline status chip rendered next to each `InputField` /
 * `ToggleField` in the Configuration form. The chip maps the
 * per-field `FieldDiff['validation']` state to a `StatusChip` variant
 * so the operator can see whether the value they typed is good
 * (silent), pending, or invalid without scrolling to the Save button.
 *
 * State-to-variant mapping:
 *   - `valid`   → renders nothing (no noise for the common path)
 *   - `pending` → neutral chip with a field-key label
 *   - `invalid` → danger chip with the validation message (or the
 *                 field-key label when no message is supplied)
 *
 * The chip pulls its label from the i18n catalog under
 * `configuration:fieldStatus.<state>` for the field's key. The exact
 * keys are documented in the W1 commit (`en/configuration.json`,
 * `es/configuration.json`).
 */

import { AlertCircle, PencilLine } from 'lucide-react';
import { StatusChip, type StatusChipVariantProps } from '@/shared/components/ui/StatusChip';
import type { FieldDiff } from '../hooks/useConfigurationDiff';
import { useT } from '@/i18n';

interface FieldStatusChipProps {
  validation: FieldDiff['validation'];
  fieldKey: keyof FieldDiff extends never ? string : string;
}

export function FieldStatusChip({ validation, fieldKey }: FieldStatusChipProps) {
  const { t } = useT();

  if (validation.state === 'valid') {
    return null;
  }

  if (validation.state === 'invalid') {
    const label = validation.message ?? t(`configuration:fieldStatus.invalid.${fieldKey}`);
    return (
      <StatusChip variant="danger" size="sm" icon={<AlertCircle className="h-3 w-3" />} ariaLabel={label}>
        {label}
      </StatusChip>
    );
  }

  // `pending` — only used when a field has a value the operator has
  // not yet changed. Useful for write-only secrets and form defaults
  // that have no configured counterpart.
  const label = t(`configuration:fieldStatus.pending.${fieldKey}`);
  return (
    <StatusChip
      variant={'neutral' as StatusChipVariantProps['variant']}
      size="sm"
      icon={<PencilLine className="h-3 w-3" />}
      ariaLabel={label}
    >
      {label}
    </StatusChip>
  );
}
