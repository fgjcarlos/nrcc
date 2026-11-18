import { Link } from 'react-router-dom';
import { ChevronRight } from 'lucide-react';
import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui';
import { formatBytes } from '@/shared/lib';
import type { BackupObservability } from '@/features/backups/services/backupService';

/**
 * Backups section of the Overview sheet (T8): scheduler status, last
 * backup, last automatic run and storage. A failed automatic run is the
 * only critical (red) state; an unscheduled scheduler is a held gate
 * (amber), and nothing received yet is "not reported".
 */

export interface BackupHealthTileProps {
  backups?: BackupObservability;
}

function formatDate(value: string | undefined, locale: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toLocaleString(locale, { dateStyle: 'medium', timeStyle: 'short' });
}

export function BackupHealthTile({ backups }: BackupHealthTileProps) {
  const { t, i18n } = useT();
  const k = (key: string) => t(`dashboard:overviewTiles.backupHealth.${key}`);
  const scheduler = backups?.scheduler;
  const latestBackup = backups?.latestBackup;
  const storage = backups?.storage;
  const nextRun = scheduler?.scheduled ? formatDate(scheduler.nextRunAt, i18n.language) : undefined;
  const lastAutomatic = formatDate(scheduler?.lastSuccessAt, i18n.language);

  const state: 'notReported' | 'withAlerts' | 'notScheduled' | 'scheduled' = !backups
    ? 'notReported'
    : scheduler?.lastError
      ? 'withAlerts'
      : !scheduler?.scheduled
        ? 'notScheduled'
        : 'scheduled';
  const variant = ({ notReported: 'neutral', withAlerts: 'danger', notScheduled: 'warning', scheduled: 'success' } as const)[state];
  const statusLabel = k(`statuses.${state}`);

  const items = [
    {
      testId: 'overview-backup-last-name',
      label: k('lastBackup'),
      value: backups ? (latestBackup?.name ?? k('never')) : k('notReported'),
      detail: latestBackup ? formatDate(latestBackup.createdAt, i18n.language) : backups ? k('noSnapshotsYet') : undefined,
    },
    {
      testId: 'overview-backup-last-automatic',
      label: k('lastAutomatic'),
      value: lastAutomatic ?? (backups ? k('noAutomaticRuns') : k('notReported')),
      designator: scheduler?.lastBackupId,
    },
    {
      testId: 'overview-backup-storage',
      label: k('storageUsed'),
      value: storage ? formatBytes(storage.totalSize) : k('notReported'),
      detail: storage ? t('dashboard:overviewTiles.backupHealth.backupCount', { count: storage.totalBackups }) : undefined,
    },
  ];

  return (
    <section
      data-testid="overview-backup-health-tile"
      className="grid gap-4 py-4 sm:py-6 lg:grid-cols-[16rem_minmax(0,1fr)] lg:gap-6"
      aria-labelledby="overview-backup-health-title"
    >
      <header className="space-y-2">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <h2 id="overview-backup-health-title" className="text-sm font-semibold text-ds-text-primary">
            {k('title')}
          </h2>
          <StatusChip variant={variant} size="sm" ariaLabel={t('dashboard:overviewTiles.backupHealth.statusAria', { state: statusLabel })}>
            {statusLabel}
          </StatusChip>
        </div>
        {backups && (
          <p className="text-xs text-ds-text-secondary">
            {scheduler?.scheduled ? k('subtitleActive') : k('subtitleInactive')}
            {nextRun && <span className="tabular-nums">{` · ${k('nextRun')} ${nextRun}`}</span>}
          </p>
        )}
        <Link
          to="/backups"
          className="inline-flex min-h-11 items-center gap-1 text-sm font-medium text-ds-on-tint-primary underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ds-accent-primary sm:min-h-0"
          data-testid="overview-backup-health-link"
        >
          {k('cta')}
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </Link>
      </header>

      <div className="min-w-0 space-y-4">
        <dl className="grid gap-x-6 sm:grid-cols-3">
          {items.map((item) => (
            <div key={item.testId} data-testid={item.testId} className="min-w-0 border-b border-ds-border-subtle py-2">
              <dt className="text-xs text-ds-text-secondary">{item.label}</dt>
              <dd className="min-w-0">
                <span className="block break-words text-sm font-semibold tabular-nums text-ds-text-primary">{item.value}</span>
                {item.detail && <span className="block text-xs tabular-nums text-ds-text-muted">{item.detail}</span>}
                {item.designator && (
                  <span className="block truncate font-mono text-xs font-semibold text-ds-text-secondary" title={item.designator}>
                    {item.designator}
                  </span>
                )}
              </dd>
            </div>
          ))}
        </dl>

        {scheduler?.lastError && (
          <p
            className="rounded-box border border-ds-danger/35 bg-ds-danger/10 px-3 py-2 text-sm text-ds-on-tint-error"
            data-testid="overview-backup-last-error"
          >
            <span className="font-semibold">{k('lastError')}</span> {scheduler.lastError}
          </p>
        )}
      </div>
    </section>
  );
}
