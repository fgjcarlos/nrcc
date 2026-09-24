import type { BackupSummary, BackupSchedulerStatus } from '@/features/backups/services';
import { formatBackupDate, formatBackupSize, getBackupDisplayName } from '@/features/backups/lib/formatters';
import { cn } from '@/shared/lib';
import { useT } from '@/i18n';

interface StorageSummary {
  totalBackups: number;
  totalSize: number;
}

interface BackupSummaryCardsProps {
  /** First item of the current backup list (or null if empty) */
  latestBackup: BackupSummary | null;
  /** Effective scheduler status (merged from API or draft config) */
  schedulerStatus: BackupSchedulerStatus & {
    nextRunAt: string;
    schedule: string;
    customSchedule?: string;
    activeSpec?: string;
  };
  /** Derived scheduler tone used for badge styling */
  schedulerTone: 'healthy' | 'muted' | 'error';
  /** Derived scheduler label text */
  schedulerLabel: string;
  /** Effective storage summary (from observability or storage query) */
  storage: StorageSummary;
}

export function BackupSummaryCards(props: BackupSummaryCardsProps) {
  const { latestBackup, schedulerStatus, schedulerTone, schedulerLabel, storage } = props;
  const { t } = useT();

  return (
    <div className="grid gap-3 md:grid-cols-3">
      {/* Last Backup Card */}
      <div className="surface-card p-5">
        <p className="text-xs uppercase tracking-[0.18em] text-base-content/45">{t('backups:lastBackupLabel')}</p>
        <div className="mt-2 text-lg font-semibold text-base-content">
          {latestBackup ? getBackupDisplayName(latestBackup) : t('backups:noBackupsShort')}
        </div>
        <p className="mt-1 text-sm text-base-content/60">
          {latestBackup
            ? formatBackupDate(latestBackup.createdAt)
            : t('backups:createFirstLocal')}
        </p>
      </div>

      {/* Next Execution Card */}
      <div className="surface-card p-5">
        <div className="flex items-center justify-between gap-3">
          <p className="text-xs uppercase tracking-[0.18em] text-base-content/45">{t('backups:nextRunLabel')}</p>
          <span
            className={cn(
              'inline-flex items-center rounded-full border px-2.5 py-1 text-[11px] font-medium',
              schedulerTone === 'healthy'
                ? 'border-success/30 bg-success/10 text-success'
                : schedulerTone === 'error'
                  ? 'border-error/30 bg-error/10 text-error'
                  : 'border-border bg-base-200/30 text-base-content/70'
            )}
          >
            {schedulerLabel}
          </span>
        </div>
        <div className="mt-2 text-lg font-semibold text-base-content">
          {schedulerStatus.nextRunAt ? formatBackupDate(schedulerStatus.nextRunAt) : t('backups:notScheduled')}
        </div>
        <p className="mt-1 text-sm text-base-content/60">
          {schedulerStatus.schedule === 'custom' && schedulerStatus.customSchedule
            ? `${t('backups:cronPrefix')} ${schedulerStatus.customSchedule}`
            : schedulerStatus.activeSpec
              ? `${t('backups:activeSpecPrefix')} ${schedulerStatus.activeSpec}`
              : t('backups:saveFrequencyToActivate')}
        </p>
      </div>

      {/* Storage Card */}
      <div className="surface-card p-5">
        <p className="text-xs uppercase tracking-[0.18em] text-base-content/45">{t('backups:storageLabel')}</p>
        <div className="mt-2 text-lg font-semibold text-base-content">{formatBackupSize(storage.totalSize)}</div>
        <p className="mt-1 text-sm text-base-content/60">{t('backups:storageBackupsDetected', { count: storage.totalBackups })}</p>
      </div>
    </div>
  );
}
