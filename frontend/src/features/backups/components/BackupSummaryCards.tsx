import type { BackupSummary, BackupSchedulerStatus } from '@/features/backups/services';
import { formatBackupDate, formatBackupSize, getBackupDisplayName } from '@/features/backups/lib/formatters';
import { StatusChip } from '@/shared/components/ui';
import { useT } from '@/i18n';

export type SchedulerState = 'failed' | 'unscheduled' | 'scheduled';

// Red only for a failed run; an unscheduled scheduler is a held gate (amber).
const STATE_VARIANT = { failed: 'danger', unscheduled: 'warning', scheduled: 'success' } as const;

interface StorageSummary {
  totalBackups: number;
  totalSize: number;
}

interface BackupSummaryCardsProps {
  /** First item of the current backup list (or null if empty) */
  latestBackup: BackupSummary | null;
  /** Effective scheduler status (merged from API or draft config) */
  schedulerStatus: BackupSchedulerStatus;
  schedulerState: SchedulerState;
  /** Effective storage summary (from observability or storage query) */
  storage: StorageSummary;
}

/** Last backup, next run and storage as one flat row of terms on the page sheet. */
export function BackupSummaryCards({ latestBackup, schedulerStatus, schedulerState, storage }: BackupSummaryCardsProps) {
  const { t, i18n } = useT();
  const date = (value: string) => formatBackupDate(value, i18n.language) ?? t('backups:dateUnavailable');
  const stateLabel = t(`backups:schedulerStatus.${schedulerState}`);
  const spec = schedulerStatus.schedule === 'custom' ? schedulerStatus.customSchedule : schedulerStatus.activeSpec;

  const items = [
    {
      key: 'last',
      label: t('backups:summary.lastBackup'),
      value: latestBackup ? getBackupDisplayName(latestBackup) : t('backups:summary.none'),
      detail: latestBackup ? date(latestBackup.createdAt) : t('backups:summary.noneDesc'),
    },
    {
      key: 'next',
      label: t('backups:summary.nextRun'),
      value: schedulerStatus.nextRunAt ? date(schedulerStatus.nextRunAt) : t('backups:summary.noRunPlanned'),
      detail: spec ? t('backups:summary.cron', { spec }) : t('backups:summary.saveToActivate'),
      mono: Boolean(spec),
    },
    {
      key: 'storage',
      label: t('backups:summary.storage'),
      value: formatBackupSize(storage.totalSize),
      detail: t('backups:summary.count', { count: storage.totalBackups }),
    },
  ];

  return (
    <section className="space-y-3 pb-4">
      <dl className="grid gap-x-6 sm:grid-cols-3">
        {items.map((item) => (
          <div key={item.key} className="min-w-0 border-b border-ds-border-subtle py-2">
            <dt className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-ds-text-secondary">
              {item.label}
              {item.key === 'next' && (
                <StatusChip variant={STATE_VARIANT[schedulerState]} size="sm" ariaLabel={t('backups:summary.statusAria', { state: stateLabel })}>
                  {stateLabel}
                </StatusChip>
              )}
            </dt>
            <dd className="min-w-0">
              <span className="block break-words text-sm font-semibold tabular-nums text-ds-text-primary">{item.value}</span>
              <span className={`block break-words text-xs tabular-nums text-ds-text-muted ${item.mono ? 'font-mono' : ''}`}>
                {item.detail}
              </span>
            </dd>
          </div>
        ))}
      </dl>

      {schedulerStatus.lastError && (
        <p className="rounded-box border border-ds-danger/35 bg-ds-danger/10 px-3 py-2 text-sm text-ds-on-tint-error">
          {schedulerStatus.lastError}
        </p>
      )}
    </section>
  );
}
