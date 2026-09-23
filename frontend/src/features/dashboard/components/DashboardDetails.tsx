import { formatBytes, cn } from '@/shared/lib';
import { formatPercent } from '@/features/dashboard/lib';
import type { BackupObservability } from '@/features/backups/services';
import type { SystemInfo } from '@/shared/types';
import { Archive, CheckCircle2, HardDrive } from 'lucide-react';
import { useT } from '@/i18n';

interface DashboardDetailsProps {
  // Restart/Open actions moved up to the RuntimeCard inside DashboardStatusCards
  // (issue #676 item 1). This component now only renders the detail cards
  // (disk breakdown, backups) that live below the metric row.
  backups?: BackupObservability;
  system?: SystemInfo;
}

function formatDate(value?: string) {
  if (!value) {
    return '--';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '--';
  }

  return date.toLocaleString();
}

function DiskUsageCard({ system }: Pick<DashboardDetailsProps, 'system'>) {
  const { t } = useT();
  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3 mb-4">
        <HardDrive className="w-5 h-5 text-body-secondary" />
        <span className="font-medium">{t('dashboard:diskUsage')}</span>
      </div>
      <div className="space-y-2">
        <div className="flex justify-between text-sm">
          <span className="text-body-secondary">
            {system?.disk.available ? formatBytes(system.disk.used) : 'Unavailable'} / {system?.disk.available ? formatBytes(system.disk.total) : '--'}
          </span>
          <span className="font-medium">{system?.disk.available ? formatPercent(system.disk.usagePercent) : '--'}</span>
        </div>
        <div className="w-full h-2 rounded-full bg-muted">
          <div
            className="h-2 transition-all duration-500 rounded-full bg-primary"
            style={{ width: `${system?.disk.available ? system.disk.usagePercent : 0}%` }}
          />
        </div>
      </div>
    </div>
  );
}

function BackupStatusCard({ backups }: Pick<DashboardDetailsProps, 'backups'>) {
  const { t } = useT();
  const scheduler = backups?.scheduler;
  const latestBackup = backups?.latestBackup;
  const recentEvent = backups?.recentEvents[0];
  const healthy = Boolean(scheduler?.scheduled && !scheduler?.lastError);
  const schedulerLabel = healthy ? t('dashboard:scheduled') : scheduler?.lastError ? t('dashboard:withAlerts') : t('dashboard:notScheduled');

  return (
    <div className="p-6 border card surface-card border-border md:col-span-2">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <Archive className="w-5 h-5 text-body-secondary" />
            <span className="font-medium">{t('dashboard:localBackups')}</span>
          </div>
          <p className="text-sm text-body-secondary">
            {scheduler?.scheduled
              ? `${t('dashboard:schedulerActive')}${scheduler.nextRunAt ? ` · ${t('dashboard:nextRun')}: ${formatDate(scheduler.nextRunAt)}` : ''}`
              : t('dashboard:schedulerInactive')}
          </p>
        </div>
        <div
          className={cn(
            'inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium',
            healthy ? 'border-success/30 bg-success/10 text-success' : 'border-border bg-base-200/30 text-base-content/70'
          )}
        >
          <CheckCircle2 className="w-3.5 h-3.5" />
          {schedulerLabel}
        </div>
      </div>

      <div className="grid gap-3 mt-5 md:grid-cols-3">
        <div className="glass-panel rounded-2xl border border-border p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/45">{t('dashboard:lastBackup')}</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{latestBackup?.name ?? t('dashboard:noBackupsShort')}</div>
          <p className="mt-1 text-sm text-body-secondary">{latestBackup ? formatDate(latestBackup.createdAt) : t('dashboard:noSnapshotsYet')}</p>
        </div>
        <div className="glass-panel rounded-2xl border border-border p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/45">{t('dashboard:lastAutomatic')}</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{formatDate(scheduler?.lastSuccessAt)}</div>
          <p className="mt-1 text-sm text-body-secondary">{scheduler?.lastBackupId ? `${t('dashboard:backup')} ${scheduler.lastBackupId}` : t('dashboard:noAutomaticRuns')}</p>
        </div>
        <div className="glass-panel rounded-2xl border border-border p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/45">{t('dashboard:storageUsed')}</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{backups ? formatBytes(backups.storage.totalSize) : '--'}</div>
          <p className="mt-1 text-sm text-body-secondary">{backups ? `${backups.storage.totalBackups} ${t('dashboard:localBackupsCount')}` : t('dashboard:loadingObservability')}</p>
        </div>
      </div>

      <div className="mt-5 glass-panel rounded-2xl border border-border p-4">
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm font-medium text-base-content">{t('dashboard:recentActivity')}</span>
          <span className="text-xs text-base-content/50">{recentEvent ? formatDate(recentEvent.occurredAt) : t('dashboard:noEvents')}</span>
        </div>
        <p className="mt-2 text-sm text-base-content">
          {recentEvent?.message ?? t('dashboard:noEventsRecorded')}
        </p>
        {scheduler?.lastError && <p className="mt-2 text-sm text-error">{scheduler.lastError}</p>}
      </div>
    </div>
  );
}

export function DashboardDetails(props: DashboardDetailsProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <DiskUsageCard system={props.system} />
      <BackupStatusCard backups={props.backups} />
    </div>
  );
}
