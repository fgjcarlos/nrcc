import { formatBytes, cn } from '@/shared/lib';
import { formatPercent } from '@/features/dashboard/lib';
import type { BackupObservability } from '@/features/backups/services';
import type { SystemInfo } from '@/shared/types';
import { Activity, Archive, CheckCircle2, ExternalLink, HardDrive, RefreshCw } from 'lucide-react';

interface DashboardDetailsProps {
  isRestarting: boolean;
  onOpenNodeRed: () => void;
  onRequestRestart: () => void;
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
  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3 mb-4">
        <HardDrive className="w-5 h-5 text-body-secondary" />
        <span className="font-medium">Disk Usage</span>
      </div>
      <div className="space-y-2">
        <div className="flex justify-between text-sm">
          <span className="text-body-secondary">
            {system ? formatBytes(system.disk.used) : '--'} / {system ? formatBytes(system.disk.total) : '--'}
          </span>
          <span className="font-medium">{system ? formatPercent(system.disk.usagePercent) : '--'}</span>
        </div>
        <div className="w-full h-2 rounded-full bg-muted">
          <div
            className="h-2 transition-all duration-500 rounded-full bg-primary"
            style={{ width: `${system?.disk.usagePercent || 0}%` }}
          />
        </div>
      </div>
    </div>
  );
}

function QuickActionsCard({
  isRestarting,
  onOpenNodeRed,
  onRequestRestart,
}: Omit<DashboardDetailsProps, 'system' | 'backups'>) {
  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3 mb-5">
        <Activity className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-semibold tracking-wide uppercase opacity-90">Quick Actions</span>
      </div>
      <div className="grid grid-cols-2 gap-2.5">
        {/* Restart */}
        <button
          onClick={onRequestRestart}
          disabled={isRestarting}
          className="group action-btn-secondary flex items-center justify-center gap-3 rounded-xl p-4"
        >
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning transition-colors group-hover:bg-warning/20">
            <RefreshCw className={cn('w-4 h-4', isRestarting && 'animate-spin')} />
          </div>
          <span className="text-base font-medium">
            {isRestarting ? 'Reiniciando…' : 'Reiniciar'}
          </span>
        </button>

        {/* Open Node-RED */}
        <button
          onClick={onOpenNodeRed}
          className="group action-btn-secondary flex items-center justify-center gap-3 rounded-xl p-4"
        >
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-info/10 text-info transition-colors group-hover:bg-info/20">
            <ExternalLink className="w-4 h-4" />
          </div>
          <span className="text-base font-medium">Abrir</span>
        </button>

        {/* Start / Stop disabled - no runtime management */}
      </div>
    </div>
  );
}

function BackupStatusCard({ backups }: Pick<DashboardDetailsProps, 'backups'>) {
  const scheduler = backups?.scheduler;
  const latestBackup = backups?.latestBackup;
  const recentEvent = backups?.recentEvents[0];
  const healthy = Boolean(scheduler?.scheduled && !scheduler?.lastError);
  const schedulerLabel = healthy ? 'Programado' : scheduler?.lastError ? 'Con alertas' : 'Sin programar';

  return (
    <div className="p-6 border card surface-card border-border md:col-span-2">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <Archive className="w-5 h-5 text-body-secondary" />
            <span className="font-medium">Backups locales</span>
          </div>
          <p className="text-sm text-body-secondary">
            {scheduler?.scheduled
              ? `Scheduler activo${scheduler.nextRunAt ? ` · próxima: ${formatDate(scheduler.nextRunAt)}` : ''}`
              : 'Scheduler sin programación activa'}
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
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/45">Último backup</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{latestBackup?.name ?? 'Sin backups'}</div>
          <p className="mt-1 text-sm text-body-secondary">{latestBackup ? formatDate(latestBackup.createdAt) : 'Todavía no hay snapshots locales'}</p>
        </div>
        <div className="glass-panel rounded-2xl border border-border p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/45">Último automático</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{formatDate(scheduler?.lastSuccessAt)}</div>
          <p className="mt-1 text-sm text-body-secondary">{scheduler?.lastBackupId ? `Backup ${scheduler.lastBackupId}` : 'Sin ejecuciones automáticas exitosas aún'}</p>
        </div>
        <div className="glass-panel rounded-2xl border border-border p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/45">Espacio ocupado</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{backups ? formatBytes(backups.storage.totalSize) : '--'}</div>
          <p className="mt-1 text-sm text-body-secondary">{backups ? `${backups.storage.totalBackups} backups locales` : 'Cargando observabilidad'}</p>
        </div>
      </div>

      <div className="mt-5 glass-panel rounded-2xl border border-border p-4">
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm font-medium text-base-content">Actividad reciente</span>
          <span className="text-xs text-base-content/50">{recentEvent ? formatDate(recentEvent.occurredAt) : 'Sin eventos'}</span>
        </div>
        <p className="mt-2 text-sm text-base-content">
          {recentEvent?.message ?? 'Todavía no hay eventos registrados para backups o scheduler.'}
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
      <QuickActionsCard
        isRestarting={props.isRestarting}
        onOpenNodeRed={props.onOpenNodeRed}
        onRequestRestart={props.onRequestRestart}
      />
      <BackupStatusCard backups={props.backups} />
    </div>
  );
}
