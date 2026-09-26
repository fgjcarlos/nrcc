import { Link } from 'react-router-dom';
import { Archive, ChevronRight, AlertTriangle, CheckCircle2 } from 'lucide-react';
import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui';
import { formatBytes, cn } from '@/shared/lib';
import type { BackupObservability } from '@/features/backups/services/backupService';

/**
 * BackupHealthTile — issue #766 slice D
 *
 * Operational decision tile for the Overview screen. Surfaces backup
 * scheduler health + last successful run + storage footprint in one
 * glance. The existing BackupStatusCard lived inside DashboardDetails
 * below the metric row; slice D promotes it to a top-level tile and
 * drops the disk card it shared the row with.
 *
 * Status semantics follow the same CVA contract as slice B/C:
 *   - scheduler.scheduled && !lastError → success
 *   - lastError set                       → warning (operator should look)
 *   - scheduler not scheduled             → danger
 */

export interface BackupHealthTileProps {
  backups?: BackupObservability;
}

function formatDate(value: string | undefined, t: (k: string) => string) {
  if (!value) return t('dashboard:overviewTiles.backupHealth.unknownDate');
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return t('dashboard:overviewTiles.backupHealth.unknownDate');
  return date.toLocaleString();
}

export function BackupHealthTile({ backups }: BackupHealthTileProps) {
  const { t } = useT();
  const scheduler = backups?.scheduler;
  const latestBackup = backups?.latestBackup;
  const storage = backups?.storage;

  const healthy = Boolean(scheduler?.scheduled && !scheduler?.lastError);
  const hasError = Boolean(scheduler?.lastError);
  const neverScheduled = !scheduler?.scheduled;

  const variant: 'success' | 'warning' | 'danger' = neverScheduled
    ? 'danger'
    : hasError
      ? 'warning'
      : healthy
        ? 'success'
        : 'warning';

  const statusLabel = neverScheduled
    ? t('dashboard:overviewTiles.backupHealth.statuses.notScheduled')
    : hasError
      ? t('dashboard:overviewTiles.backupHealth.statuses.withAlerts')
      : t('dashboard:overviewTiles.backupHealth.statuses.scheduled');

  return (
    <article
      data-testid="overview-backup-health-tile"
      className="card surface-card border border-border p-6"
      aria-labelledby="overview-backup-health-title"
    >
      <header className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <Archive className="h-6 w-6 text-base-content/70" aria-hidden="true" />
          <div className="min-w-0">
            <h2 id="overview-backup-health-title" className="text-base font-semibold text-base-content">
              {t('dashboard:overviewTiles.backupHealth.title')}
            </h2>
            <p className="mt-0.5 text-xs text-base-content/65">
              {scheduler?.scheduled
                ? `${t('dashboard:overviewTiles.backupHealth.subtitleActive')}${scheduler.nextRunAt ? ` · ${t('dashboard:overviewTiles.backupHealth.nextRun')}: ${formatDate(scheduler.nextRunAt, t)}` : ''}`
                : t('dashboard:overviewTiles.backupHealth.subtitleInactive')}
            </p>
          </div>
        </div>
        <StatusChip
          variant={variant}
          size="sm"
          ariaLabel={t('dashboard:overviewTiles.backupHealth.statusAria', { state: statusLabel })}
        >
          {hasError ? (
            <AlertTriangle className="h-3 w-3" aria-hidden="true" />
          ) : healthy ? (
            <CheckCircle2 className="h-3 w-3" aria-hidden="true" />
          ) : null}
          {statusLabel}
        </StatusChip>
      </header>

      <div className="mt-5 grid grid-cols-3 gap-3">
        <div
          className={cn(
            'rounded-xl border border-border/60 bg-base-200/30 px-3 py-3',
            'text-xs',
          )}
          data-testid="overview-backup-last-name"
        >
          <div className="text-base-content/45 uppercase tracking-[0.18em]">
            {t('dashboard:overviewTiles.backupHealth.lastBackup')}
          </div>
          <div className="mt-1.5 truncate text-base font-semibold text-base-content">
            {latestBackup?.name ?? t('dashboard:overviewTiles.backupHealth.never')}
          </div>
          <p className="mt-0.5 text-base-content/55">
            {latestBackup ? formatDate(latestBackup.createdAt, t) : t('dashboard:overviewTiles.backupHealth.noSnapshotsYet')}
          </p>
        </div>
        <div
          className="rounded-xl border border-border/60 bg-base-200/30 px-3 py-3 text-xs"
          data-testid="overview-backup-last-automatic"
        >
          <div className="text-base-content/45 uppercase tracking-[0.18em]">
            {t('dashboard:overviewTiles.backupHealth.lastAutomatic')}
          </div>
          <div className="mt-1.5 text-base font-semibold text-base-content">
            {formatDate(scheduler?.lastSuccessAt, t)}
          </div>
          <p className="mt-0.5 text-base-content/55">
            {scheduler?.lastBackupId
              ? `${t('dashboard:overviewTiles.backupHealth.backupIdPrefix')} ${scheduler.lastBackupId}`
              : t('dashboard:overviewTiles.backupHealth.noAutomaticRuns')}
          </p>
        </div>
        <div
          className="rounded-xl border border-border/60 bg-base-200/30 px-3 py-3 text-xs"
          data-testid="overview-backup-storage"
        >
          <div className="text-base-content/45 uppercase tracking-[0.18em]">
            {t('dashboard:overviewTiles.backupHealth.storageUsed')}
          </div>
          <div className="mt-1.5 text-base font-semibold text-base-content">
            {storage ? formatBytes(storage.totalSize) : t('dashboard:overviewTiles.backupHealth.unknownDate')}
          </div>
          <p className="mt-0.5 text-base-content/55">
            {storage
              ? `${storage.totalBackups} ${t('dashboard:overviewTiles.backupHealth.countSuffix')}`
              : t('dashboard:overviewTiles.backupHealth.unknownDate')}
          </p>
        </div>
      </div>

      {scheduler?.lastError && (
        <p
          className="mt-4 rounded-xl border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          data-testid="overview-backup-last-error"
        >
          {scheduler.lastError}
        </p>
      )}

      <Link
        to="/backups"
        className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-accent transition-colors hover:text-accent-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-1 focus-visible:ring-offset-base-100"
        data-testid="overview-backup-health-link"
      >
        {t('dashboard:overviewTiles.backupHealth.cta')}
        <ChevronRight className="h-4 w-4" aria-hidden="true" />
      </Link>
    </article>
  );
}
