import { Archive, CircleAlert, HardDrive, LoaderCircle, Sparkles, Trash2, Download, RotateCcw } from 'lucide-react';
import type { BackupSummary } from '@/features/backups/services';
import { formatBackupDate, formatBackupSize, getBackupDisplayName, getBackupSummary } from '@/features/backups/lib/formatters';
import { cn } from '@/shared/lib';
import { StateContainer } from '@/shared/components';
import { UI_COPY } from '@/shared/constants/uiCopy';

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
  onPageChange: (page: number) => void;
  /** True when create backup mutation is pending (for empty-state button) */
  isCreating: boolean;
  onCreateBackup: () => void;
}

const typeLabels: Record<BackupSummary['type'], string> = {
  manual: 'Manual',
  auto: 'Auto',
  'pre-restore': 'Pre-restore',
};

const typeStyles: Record<BackupSummary['type'], string> = {
  manual: 'bg-success/15 text-success-content',
  auto: 'bg-info/15 text-info-content',
  'pre-restore': 'bg-warning/15 text-warning-content',
};

export function BackupListSection(props: BackupListSectionProps) {
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
    onPageChange,
    isCreating,
    onCreateBackup,
  } = props;

  const totalPages = Math.ceil(total / 10);
  const canPrev = page > 1;
  const canNext = page < totalPages;

  return (
    <div className="surface-card p-6">
      <div className="mb-4 flex items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold text-base-content">Available Backups</h2>
          <p className="text-sm text-base-content/65">Download, restore or delete local snapshots</p>
        </div>
        {isLoading && <span className="text-sm text-base-content/60">{UI_COPY.loading}</span>}
      </div>

      <StateContainer
        isLoading={isLoading}
        isError={isError}
        isEmpty={backups.length === 0}
        loadingSlot={
          <div className="glass-panel rounded-2xl border border-border p-10 text-center">
            <LoaderCircle className="mx-auto mb-3 h-10 w-10 animate-spin text-base-content/40" />
            <p className="font-medium text-base-content">{UI_COPY.loadingBackups}</p>
            <p className="text-sm text-base-content/65">{UI_COPY.readingBackups}</p>
          </div>
        }
        errorSlot={
          <div className="rounded-2xl border border-error/20 bg-error/8 p-6 text-sm text-base-content">
            <div className="flex items-center gap-2 font-medium text-error">
              <CircleAlert className="h-4 w-4" />
              {UI_COPY.failedToLoadBackups}
            </div>
            <p className="mt-2 text-base-content/70">{UI_COPY.retryLoadBackups}</p>
          </div>
        }
        emptySlot={
          <div className="glass-panel rounded-2xl border border-dashed border-border p-10 text-center">
            <HardDrive className="mx-auto mb-3 h-10 w-10 text-base-content/40" />
            <p className="font-medium text-base-content">{UI_COPY.noBackupsYet}</p>
            <p className="text-sm text-base-content/65">{UI_COPY.createBackupDesc}</p>
            <button onClick={onCreateBackup} disabled={isCreating} className="action-btn-primary mt-4">
              {isCreating && <LoaderCircle className="h-4 w-4 animate-spin" />}
              {UI_COPY.add} {UI_COPY.createFirstBackup}
            </button>
          </div>
        }
      >
        <div className="space-y-4">
          <div className="overflow-x-auto">
            <div className="surface-panel min-w-[760px] overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-border text-left text-sm text-base-content/60">
                    <th className="py-3 pr-4">{UI_COPY.type}</th>
                    <th className="py-3 pr-4">{UI_COPY.name}</th>
                    <th className="py-3 pr-4">{UI_COPY.date}</th>
                    <th className="py-3 pr-4">{UI_COPY.files}</th>
                    <th className="py-3 pr-4">{UI_COPY.size}</th>
                    <th className="py-3 pr-4">{UI_COPY.actions}</th>
                  </tr>
                </thead>
                <tbody>
                  {backups.map((backup) => (
                    <tr
                      key={backup.id}
                      className={cn(
                        'border-b border-border text-sm transition-colors hover:bg-base-200/25',
                        selectedBackupId === backup.id && 'bg-primary/5'
                      )}
                    >
                      {/* Type Badge */}
                      <td className="py-3 pr-4">
                        <button
                          onClick={() => onSelect(backup.id)}
                          title={getBackupSummary(backup, typeLabels)}
                          className={cn(
                            'inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs font-medium transition-colors hover:opacity-90',
                            typeStyles[backup.type]
                          )}
                        >
                          {backup.type === 'auto' ? <Sparkles className="h-3 w-3" /> : <Archive className="h-3 w-3" />}
                          {typeLabels[backup.type]}
                        </button>
                      </td>

                      {/* Name */}
                      <td className="py-3 pr-4 text-base-content">
                        <button
                          onClick={() => onSelect(backup.id)}
                          className="text-left transition-colors hover:text-primary"
                        >
                          <div className="font-medium">{getBackupDisplayName(backup)}</div>
                        </button>
                      </td>

                      {/* Date */}
                      <td className="py-3 pr-4 text-base-content/70">{formatBackupDate(backup.createdAt)}</td>

                      {/* File Count */}
                      <td className="py-3 pr-4 text-base-content/70">
                        {backup.fileCount != null ? String(backup.fileCount) : '--'}
                      </td>

                      {/* Size */}
                      <td className="py-3 pr-4 font-medium text-base-content">{formatBackupSize(backup.totalSize)}</td>

                      {/* Actions */}
                      <td className="py-3 pr-4">
                        <div className="flex items-center gap-2">
                          <button
                            onClick={() => onDownload(backup)}
                            disabled={isActionPending && pendingActionId === backup.id}
                            title={UI_COPY.download}
                            className="icon-button"
                          >
                            {isActionPending && pendingActionId === backup.id ? (
                              <LoaderCircle className="h-4 w-4 animate-spin" />
                            ) : (
                              <Download className="h-4 w-4" />
                            )}
                          </button>
                          <button
                            onClick={() => onRestore(backup)}
                            disabled={isActionPending && pendingActionId === backup.id}
                            title={UI_COPY.restore}
                            className="icon-button"
                          >
                            {isActionPending && pendingActionId === backup.id ? (
                              <LoaderCircle className="h-4 w-4 animate-spin" />
                            ) : (
                              <RotateCcw className="h-4 w-4" />
                            )}
                          </button>
                          <button
                            onClick={() => onDelete(backup)}
                            disabled={isActionPending && pendingActionId === backup.id}
                            title={UI_COPY.delete}
                            className="icon-button text-error hover:text-error/80"
                          >
                            {isActionPending && pendingActionId === backup.id ? (
                              <LoaderCircle className="h-4 w-4 animate-spin" />
                            ) : (
                              <Trash2 className="h-4 w-4" />
                            )}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <div className="text-sm text-base-content/60">
                {UI_COPY.pageOf(page, totalPages)}
              </div>
              <div className="flex gap-2">
                <button
                  onClick={() => onPageChange(page - 1)}
                  disabled={!canPrev}
                  className="btn-secondary btn-sm"
                >
                  {UI_COPY.previousPage}
                </button>
                <button
                  onClick={() => onPageChange(page + 1)}
                  disabled={!canNext}
                  className="btn-secondary btn-sm"
                >
                  {UI_COPY.nextPage}
                </button>
              </div>
            </div>
          )}
        </div>
      </StateContainer>
    </div>
  );
}
