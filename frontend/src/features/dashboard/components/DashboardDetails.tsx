import { formatBytes, cn } from '@/shared/lib';
import { formatPercent } from '@/features/dashboard/lib';
import type { BackupObservability } from '@/features/backups/services';
import type { RuntimeInfo, SystemInfo } from '@/shared/types';
import { Activity, Archive, CheckCircle2, ExternalLink, HardDrive, Play, RefreshCw, Square } from 'lucide-react';

interface DashboardDetailsProps {
  isRestarting: boolean;
  isStartStopping: boolean;
  onStartNodeRed: () => void;
  onStopNodeRed: () => void;
  onOpenNodeRed: () => void;
  onRequestRestart: () => void;
  backups?: BackupObservability;
  runtime?: RuntimeInfo;
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
  isStartStopping,
  onStartNodeRed,
  onStopNodeRed,
  runtime,
}: Omit<DashboardDetailsProps, 'system'>) {
  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3 mb-5">
        <Activity className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-semibold tracking-wide uppercase opacity-90">Quick Actions</span>
      </div>
       <div className="flex flex-col items-center gap-2.5">
          {/* Restart: Warning action (caution/amber) */}
          <button
            onClick={onRequestRestart}
            disabled={isRestarting}
            className="w-full max-w-xs gap-3 font-medium transition-colors duration-150 btn btn-warning btn-sm h-11 quick-action-btn"
          >
            <RefreshCw className={cn('flex-shrink-0 w-4 h-4', isRestarting && 'animate-spin')} />
            <span>{isRestarting ? 'Reiniciando…' : 'Reiniciar Node-RED'}</span>
          </button>

           {/* Open: Info action - navigation with background */}
           <button 
             onClick={onOpenNodeRed} 
             className="w-full max-w-xs gap-3 font-medium transition-colors duration-150 btn btn-info btn-sm h-11 quick-action-btn"
           >
             <ExternalLink className="flex-shrink-0 w-4 h-4" />
             <span>Abrir Node-RED</span>
           </button>

          {/* Start: Success action (positive/teal) */}
          {runtime?.status === 'stopped' && (
            <button
              onClick={onStartNodeRed}
              disabled={isStartStopping}
              className="w-full gap-3 font-medium transition-colors duration-150 max-w-sx btn btn-success btn-sm h-11 quick-action-btn"
            >
              <Play className="flex-shrink-0 w-4 h-4" />
              <span>{isStartStopping ? 'Iniciando…' : 'Arrancar'}</span>
            </button>
          )}

          {/* Stop: Error action (destructive/red) */}
          {runtime?.status === 'running' && (
            <button
              onClick={onStopNodeRed}
              disabled={isStartStopping}
              className="w-full max-w-xs gap-3 font-medium transition-colors duration-150 btn btn-error btn-sm h-11 quick-action-btn"
            >
              <Square className="flex-shrink-0 w-4 h-4" />
              <span>{isStartStopping ? 'Deteniendo…' : 'Detener Node-RED'}</span>
            </button>
          )}
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
            healthy ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-border bg-base-200/30 text-base-content/70'
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

      <div className="mt-5 rounded-2xl border border-border/70 bg-base-200/20 p-4">
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
        isStartStopping={props.isStartStopping}
        onStartNodeRed={props.onStartNodeRed}
        onStopNodeRed={props.onStopNodeRed}
        runtime={props.runtime}
      />
      <BackupStatusCard backups={props.backups} />
    </div>
  );
}
