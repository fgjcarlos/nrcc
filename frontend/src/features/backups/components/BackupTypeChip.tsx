import { Archive, Clock, History } from 'lucide-react';
import { StatusChip } from '@/shared/components/ui';
import { useT } from '@/i18n';
import type { BackupSummary } from '@/features/backups/services';

const ICONS = { manual: Archive, auto: Clock, 'pre-restore': History } as const;

/** Backup type as a neutral chip: a type is not a state, so no conductor colour (One Job Rule). */
export function BackupTypeChip({ type }: { type: BackupSummary['type'] }) {
  const { t } = useT();
  const Icon = ICONS[type];
  return (
    <StatusChip variant="neutral" size="sm" icon={<Icon className="h-3 w-3" />} className="whitespace-nowrap">
      {t(`backups:types.${type}`)}
    </StatusChip>
  );
}
