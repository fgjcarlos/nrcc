import { Download, HardDrive, LoaderCircle, RotateCcw, Trash2 } from 'lucide-react';
import type { BackupSummary } from '@/features/backups/services';
import { formatBackupDate, formatBackupSize, getBackupDisplayName } from '@/features/backups/lib/formatters';
import { cn } from '@/shared/lib';
import { StateContainer } from '@/shared/components';
import { EmptyState, ErrorState } from '@/shared/components/feedback';
import { Button, Table } from '@/shared/components/ui';
import { useT } from '@/i18n';
import { BackupTypeChip } from './BackupTypeChip';

interface BackupListSectionProps {
  backups: BackupSummary[];
  isLoading: boolean;
  isError: boolean;
  /** Currently selected backup ID (or null) */
  selectedBackupId: string | null;
  /** Total number of backups for pagination calculation */
  total: number;
  page: number;
  /** Whether a restore or delete operation is in progress */
  isActionPending: boolean;
  /** ID of the backup currently undergoing restore or delete (or null) */
  pendingActionId: string | null;
  onSelect: (id: string) => void;
  onDownload: (backup: BackupSummary) => void;
  onRestore: (backup: BackupSummary) => void;
  onDelete: (backup: BackupSummary) => void;
  onRetry: () => void;
  onPageChange: (page: number) => void;
  /** True when create backup mutation is pending (for empty-state button) */
  isCreating: boolean;
  onCreateBackup: () => void;
}

// 44px touch target on phones, 36px from sm up.
const iconButton = 'min-h-11 min-w-11 justify-center px-0 sm:min-h-9 sm:min-w-9';
const spinner = <LoaderCircle className="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />;

export function BackupListSection(props: BackupListSectionProps) {
  const { t, i18n } = useT();
  const {
    backups,
    isLoading,
    isError,
    selectedBackupId,
    total,
    page,
    isActionPending,
    pendingActionId,
    onSelect,
    onDownload,
    onRestore,
    onDelete,
    onRetry,
    onPageChange,
    isCreating,
    onCreateBackup,
  } = props;

  const totalPages = Math.ceil(total / 10);
  const canPrev = page > 1;
  const canNext = page < totalPages;

  return (
    <section aria-labelledby="backups-list-title" className="space-y-3 py-4">
      <div className="space-y-1">
        <h2 id="backups-list-title" className="text-sm font-semibold text-ds-text-primary">
          {t('backups:list.title')}
        </h2>
        <p className="text-xs text-ds-text-secondary">{t('backups:list.desc')}</p>
      </div>

      <StateContainer
        isLoading={isLoading}
        isError={isError}
        isEmpty={backups.length === 0}
        loadingLabel={t('backups:loading')}
        errorSlot={
          <ErrorState title={t('backups:failedToLoad')} description={t('backups:retryLoad')} onRetry={onRetry} />
        }
        emptySlot={
          <EmptyState
            icon={<HardDrive className="h-6 w-6" />}
            title={t('backups:noBackupsYet')}
            description={t('backups:createFirstDesc')}
            actionLabel={t('backups:createFirst')}
            onAction={() => {
              if (!isCreating) onCreateBackup();
            }}
          />
        }
      >
        <div className="space-y-3">
          <Table stacked caption={t('backups:table.caption')}>
            <thead>
              <tr>
                <th>{t('backups:table.type')}</th>
                <th>{t('backups:table.name')}</th>
                <th>{t('backups:table.date')}</th>
                <th className="text-right">{t('backups:table.files')}</th>
                <th className="text-right">{t('backups:table.size')}</th>
                <th className="text-right">{t('backups:table.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {backups.map((backup) => {
                const date = formatBackupDate(backup.createdAt, i18n.language) ?? t('backups:dateUnavailable');
                const busy = isActionPending && pendingActionId === backup.id;
                const selected = selectedBackupId === backup.id;
                return (
                  <tr key={backup.id} className={cn(selected && 'bg-ds-accent-primary/10')}>
                    <td data-label={t('backups:table.type')}>
                      <BackupTypeChip type={backup.type} />
                    </td>
                    <td data-label={t('backups:table.name')}>
                      <button
                        type="button"
                        onClick={() => onSelect(backup.id)}
                        aria-expanded={selected}
                        className="min-h-6 break-words text-left font-medium text-ds-on-tint-primary underline-offset-4 hover:underline"
                      >
                        {getBackupDisplayName(backup)}
                      </button>
                    </td>
                    <td data-label={t('backups:table.date')} className="whitespace-nowrap tabular-nums text-ds-text-secondary">
                      {date}
                    </td>
                    <td data-label={t('backups:table.files')} className="text-right tabular-nums text-ds-text-secondary">
                      {String(backup.fileCount)}
                    </td>
                    <td data-label={t('backups:table.size')} className="whitespace-nowrap text-right tabular-nums">
                      {formatBackupSize(backup.totalSize)}
                    </td>
                    <td data-label={t('backups:table.actions')} className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          variant="ghost"
                          onClick={() => onDownload(backup)}
                          disabled={busy}
                          title={t('backups:actions.download')}
                          aria-label={t('backups:actions.downloadAria', { date })}
                          className={iconButton}
                        >
                          {busy ? spinner : <Download className="h-4 w-4" aria-hidden="true" />}
                        </Button>
                        <Button
                          variant="ghost"
                          onClick={() => onRestore(backup)}
                          disabled={busy}
                          title={t('backups:actions.restore')}
                          aria-label={t('backups:actions.restoreAria', { date })}
                          className={iconButton}
                        >
                          {busy ? spinner : <RotateCcw className="h-4 w-4" aria-hidden="true" />}
                        </Button>
                        <Button
                          variant="ghost"
                          onClick={() => onDelete(backup)}
                          disabled={busy}
                          title={t('backups:actions.delete')}
                          aria-label={t('backups:actions.deleteAria', { date })}
                          className={cn(iconButton, 'text-ds-on-tint-error')}
                        >
                          {busy ? spinner : <Trash2 className="h-4 w-4" aria-hidden="true" />}
                        </Button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </Table>

          {totalPages > 1 && (
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="text-xs tabular-nums text-ds-text-secondary">
                {t('common:pageOf', { current: page, total: totalPages })}
              </div>
              <div className="flex gap-2">
                <Button variant="secondary" size="sm" onClick={() => onPageChange(page - 1)} disabled={!canPrev}>
                  {t('common:previousPage')}
                </Button>
                <Button variant="secondary" size="sm" onClick={() => onPageChange(page + 1)} disabled={!canNext}>
                  {t('common:nextPage')}
                </Button>
              </div>
            </div>
          )}
        </div>
      </StateContainer>
    </section>
  );
}
