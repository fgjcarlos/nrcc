import { Link } from 'react-router-dom';
import { ChevronRight, Cpu, ExternalLink, MemoryStick, RefreshCw, Server } from 'lucide-react';
import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui';
import { cn, formatBytes } from '@/shared/lib';
import { formatPercent } from '@/features/dashboard/lib';
import type { HostStatus, RuntimeInfo, SystemInfo } from '@/shared/types';
import type { DashboardContainerStatus } from '../../types';

/**
 * NodeRedHealthTile — issue #766 slice D
 *
 * Operational decision tile for the Overview screen. Folds the
 * SystemHealthCard (host readiness + issue list), the runtime card
 * (process state + restart/open actions) and a compact resource
 * indicator (CPU/Memory/Disk) into one tile so the operator can
 * answer "is Node-RED healthy?" and "should I restart?" at a glance.
 *
 * The resource indicator is intentionally compact (three percentages)
 * rather than full sparklines; chart detail lives on the dedicated
 * resource panels below the tile.
 */

export interface NodeRedHealthTileProps {
  host?: HostStatus;
  runtime?: RuntimeInfo;
  system?: SystemInfo;
  container?: DashboardContainerStatus | null;
  inDocker: boolean;
  isRestarting: boolean;
  onRequestRestart: () => void;
  onOpenNodeRed: () => void;
}

function scopeLabel(scope: SystemInfo['resourceScope'] | undefined, t: (k: string) => string) {
  if (scope === 'container') return t('dashboard:overviewTiles.nodeRedHealth.scopeContainer');
  if (scope === 'host') return t('dashboard:overviewTiles.nodeRedHealth.scopeHost');
  return t('dashboard:overviewTiles.nodeRedHealth.scopeUnavailable');
}

function getDeploymentLabel(
  mode: HostStatus['nodeRed']['mode'] | undefined,
  t: (k: string) => string,
) {
  switch (mode) {
    case 'native':
      return t('dashboard:overviewTiles.nodeRedHealth.deploymentNative');
    case 'docker':
      return t('dashboard:overviewTiles.nodeRedHealth.deploymentDocker');
    case 'none':
      return t('dashboard:overviewTiles.nodeRedHealth.deploymentNone');
    default:
      return t('dashboard:overviewTiles.nodeRedHealth.deploymentUnknown');
  }
}

export function NodeRedHealthTile({
  host,
  runtime,
  system,
  container,
  inDocker,
  isRestarting,
  onRequestRestart,
  onOpenNodeRed,
}: NodeRedHealthTileProps) {
  const { t } = useT();
  const runtimeStatus = runtime?.status ?? (host?.nodeRed.running ? 'running' : 'unknown');
  const canRestart = Boolean(host?.nodeRed?.detected);

  const hostReady = Boolean(host?.ready);
  const issues = host && !host.ready
    ? [
        !host.nodejs.installed ? t('dashboard:hostWarning.nodejsMissing') : '',
        !host.nodeRed.detected ? t('dashboard:hostWarning.nodeRedNotDetected') : '',
        !host.settings.writable ? t('dashboard:hostWarning.settingsNotWritable') : '',
      ].filter(Boolean)
    : [];

  const overallVariant: 'success' | 'warning' | 'danger' = !hostReady
    ? 'danger'
    : runtimeStatus !== 'running'
      ? 'warning'
      : 'success';

  return (
    <article
      data-testid="overview-node-red-health-tile"
      className="card surface-card border border-border p-6"
      aria-labelledby="overview-node-red-health-title"
    >
      <header className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <Server className="h-6 w-6 text-base-content/70" aria-hidden="true" />
          <div className="min-w-0">
            <h2 id="overview-node-red-health-title" className="text-base font-semibold text-base-content">
              {t('dashboard:overviewTiles.nodeRedHealth.title')}
            </h2>
            <p className="mt-0.5 text-xs text-base-content/65">
              {inDocker
                ? `${t('dashboard:overviewTiles.nodeRedHealth.subtitleRuntime')}: ${runtimeStatus}`
                : getDeploymentLabel(host?.nodeRed.mode, t)}
            </p>
          </div>
        </div>
        <StatusChip
          variant={overallVariant}
          size="sm"
          ariaLabel={t('dashboard:overviewTiles.nodeRedHealth.statusAria', {
            state: runtimeStatus,
          })}
        >
          {runtimeStatus}
        </StatusChip>
      </header>

      {!hostReady && issues.length > 0 && (
        <ul
          className="mt-3 space-y-1 rounded-xl border border-warning/30 bg-warning/10 px-3 py-2 text-xs text-warning"
          data-testid="overview-node-red-issues"
        >
          {issues.map((issue) => (
            <li key={issue}>• {issue}</li>
          ))}
        </ul>
      )}

      <div className="mt-5 grid grid-cols-3 gap-3" data-testid="overview-node-red-resources">
        <div className="rounded-xl border border-border/60 bg-base-200/30 px-3 py-3 text-xs">
          <div className="flex items-center gap-1.5 text-base-content/45 uppercase tracking-[0.18em]">
            <Cpu className="h-3 w-3" aria-hidden="true" />
            {t('dashboard:overviewTiles.nodeRedHealth.cpu')}
          </div>
          <div className="mt-1.5 text-base font-semibold text-base-content">
            {system?.cpu.available ? formatPercent(system.cpu.usage) : '—'}
          </div>
          <p className="mt-0.5 text-base-content/55">
            {system?.cpu.available
              ? `${system.cpu.cores} ${t('dashboard:overviewTiles.nodeRedHealth.cores')}`
              : scopeLabel(system?.resourceScope, t)}
          </p>
        </div>
        <div className="rounded-xl border border-border/60 bg-base-200/30 px-3 py-3 text-xs">
          <div className="flex items-center gap-1.5 text-base-content/45 uppercase tracking-[0.18em]">
            <MemoryStick className="h-3 w-3" aria-hidden="true" />
            {t('dashboard:overviewTiles.nodeRedHealth.memory')}
          </div>
          <div className="mt-1.5 text-base font-semibold text-base-content">
            {system?.memory.available ? formatPercent(system.memory.usagePercent) : '—'}
          </div>
          <p className="mt-0.5 text-base-content/55">
            {system?.memory.available
              ? `${formatBytes(system.memory.used)} / ${formatBytes(system.memory.total)}`
              : scopeLabel(system?.resourceScope, t)}
          </p>
        </div>
        <div className="rounded-xl border border-border/60 bg-base-200/30 px-3 py-3 text-xs">
          <div className="flex items-center gap-1.5 text-base-content/45 uppercase tracking-[0.18em]">
            {t('dashboard:overviewTiles.nodeRedHealth.disk')}
          </div>
          <div className="mt-1.5 text-base font-semibold text-base-content">
            {system?.disk.available ? formatPercent(system.disk.usagePercent) : '—'}
          </div>
          <p className="mt-0.5 text-base-content/55">
            {system?.disk.available
              ? `${formatBytes(system.disk.used)} / ${formatBytes(system.disk.total)}`
              : scopeLabel(system?.resourceScope, t)}
          </p>
        </div>
      </div>

      <div className="mt-5 grid grid-cols-2 gap-2.5">
        <button
          type="button"
          aria-label={isRestarting
            ? t('dashboard:runtimeCard.restartingButton')
            : t('dashboard:runtimeCard.restartButton')}
          onClick={onRequestRestart}
          disabled={isRestarting || !canRestart}
          data-testid="overview-node-red-restart"
          className={cn(
            'group action-btn-secondary flex items-center justify-center gap-3 rounded-xl p-4',
            'disabled:cursor-not-allowed disabled:opacity-50',
          )}
        >
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning transition-colors group-hover:bg-warning/20">
            <RefreshCw className={cn('w-4 h-4', isRestarting && 'animate-spin')} />
          </div>
          <span className="text-base font-medium">
            {isRestarting ? t('dashboard:runtimeCard.restartingButton') : t('dashboard:runtimeCard.restartButton')}
          </span>
        </button>
        <button
          type="button"
          aria-label={t('dashboard:runtimeCard.openButton')}
          onClick={onOpenNodeRed}
          data-testid="overview-node-red-open"
          className="group action-btn-secondary flex items-center justify-center gap-3 rounded-xl p-4"
        >
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-info/10 text-info transition-colors group-hover:bg-info/20">
            <ExternalLink className="w-4 h-4" />
          </div>
          <span className="text-base font-medium">{t('dashboard:runtimeCard.openButton')}</span>
        </button>
      </div>

      <Link
        to="/configuration"
        className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-accent transition-colors hover:text-accent-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-1 focus-visible:ring-offset-base-100"
        data-testid="overview-node-red-link"
      >
        {t('dashboard:overviewTiles.nodeRedHealth.cta')}
        <ChevronRight className="h-4 w-4" aria-hidden="true" />
      </Link>

      {container?.image && inDocker && (
        <p className="mt-3 truncate text-xs text-base-content/55" title={container.image}>
          {container.image}
        </p>
      )}
    </article>
  );
}
