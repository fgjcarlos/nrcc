import { X } from 'lucide-react';
import type { BackupSummary } from '@/features/backups/services';
import { formatBackupDate, formatBackupSize, getBackupDisplayName } from '@/features/backups/lib/formatters';
import { StateContainer } from '@/shared/components';
import { EmptyState } from '@/shared/components/feedback';
import { Button, Table } from '@/shared/components/ui';
import { useT } from '@/i18n';
import { BackupTypeChip } from './BackupTypeChip';

interface BackupDetail {
  files: Array<{
    path: string;
    size: number;
    checksum: string;
  }>;
}

interface BackupDetailSectionProps {
  backup: BackupSummary;
  detail: BackupDetail | undefined;
  isLoading: boolean;
  onClose: () => void;
}

export function BackupDetailSection(props: BackupDetailSectionProps) {
  const { backup, detail, isLoading, onClose } = props;
  const { t, i18n } = useT();

  if (!detail && !isLoading) {
    return null;
  }

  const files = detail?.files ?? [];

  return (
    <section aria-labelledby="backup-detail-title" className="space-y-3 py-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-1">
          <h2 id="backup-detail-title" className="text-sm font-semibold text-ds-text-primary">
            {t('backups:detail.title')} · <span className="break-words">{getBackupDisplayName(backup)}</span>
          </h2>
          <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs tabular-nums text-ds-text-secondary">
            <BackupTypeChip type={backup.type} />
            <span>{formatBackupDate(backup.createdAt, i18n.language) ?? t('backups:dateUnavailable')}</span>
            <span aria-hidden="true">·</span>
            <span>{formatBackupSize(backup.totalSize)}</span>
          </p>
        </div>
        <Button
          variant="ghost"
          onClick={onClose}
          aria-label={t('backups:detail.close')}
          className="min-h-11 min-w-11 shrink-0 justify-center px-0 md:min-h-9 md:min-w-9"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </Button>
      </div>

      <StateContainer
        isLoading={isLoading && !detail}
        isError={false}
        isEmpty={files.length === 0}
        loadingLabel={t('backups:detail.loading')}
        emptySlot={<EmptyState title={t('backups:detail.empty')} />}
      >
        <Table stacked caption={t('backups:detail.title')}>
          <thead>
            <tr>
              <th>{t('backups:detail.path')}</th>
              <th className="text-right">{t('backups:detail.size')}</th>
              <th>{t('backups:detail.checksum')}</th>
            </tr>
          </thead>
          <tbody>
            {files.map((file) => (
              <tr key={file.path}>
                <td data-label={t('backups:detail.path')} className="break-all font-mono text-xs">
                  {file.path}
                </td>
                <td data-label={t('backups:detail.size')} className="whitespace-nowrap text-right tabular-nums">
                  {formatBackupSize(file.size)}
                </td>
                <td
                  data-label={t('backups:detail.checksum')}
                  className="font-mono text-xs text-ds-text-secondary"
                  title={file.checksum}
                >
                  {file.checksum.slice(0, 16)}…
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
      </StateContainer>
    </section>
  );
}
