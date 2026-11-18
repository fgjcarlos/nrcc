import { Link } from 'react-router-dom';
import { ChevronRight, ExternalLink, RefreshCw } from 'lucide-react';
import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui';
import { cn, formatBytes } from '@/shared/lib';
import type { HostStatus, RuntimeInfo, SystemInfo } from '@/shared/types';
import type { DashboardContainerStatus } from '../../types';
import { runtimeKey } from '../../lib';

/**
 * Node-RED section of the Overview sheet (T8): runtime status, the host
 * issues that block it, resource usage and the two runtime actions. Opening
 * the editor is the everyday action; Restart is quieter and stays behind its
 * confirmation dialog.
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

function getDeploymentLabel(mode: HostStatus['nodeRed']['mode'] | undefined, t: (k: string) => string) {
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

interface Metric {
  key: 'cpu' | 'memory' | 'disk';
  percent?: number;
  detail?: string;
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
  const status = runtimeKey(runtime?.status ?? (host?.nodeRed.running ? 'running' : undefined));
  const canRestart = Boolean(host?.nodeRed?.detected);
  const version = host?.nodeRed.version || system?.nodeRedVersion;

  const hostReady = Boolean(host?.ready);
  const issues = host && !host.ready
    ? [
        !host.nodejs.installed ? t('dashboard:hostWarning.nodejsMissing') : '',
        !host.nodeRed.detected ? t('dashboard:hostWarning.nodeRedNotDetected') : '',
        !host.settings.writable ? t('dashboard:hostWarning.settingsNotWritable') : '',
      ].filter(Boolean)
    : [];

  // Red is kept for a host that cannot run Node-RED; unknown stays neutral.
  const overallVariant: 'success' | 'warning' | 'danger' | 'neutral' = host && !hostReady
    ? 'danger'
    : status === 'running'
      ? 'success'
      : status === 'unknown'
        ? 'neutral'
        : 'warning';
  const statusLabel = t(`dashboard:overviewTiles.nodeRedHealth.statuses.${status}`);

  const metrics: Metric[] = [
    {
      key: 'cpu',
      percent: system?.cpu.available ? system.cpu.usage : undefined,
      detail: system?.cpu.available ? `${system.cpu.cores} ${t('dashboard:overviewTiles.nodeRedHealth.cores')}` : undefined,
    },
    {
      key: 'memory',
      percent: system?.memory.available ? system.memory.usagePercent : undefined,
      detail: system?.memory.available ? `${formatBytes(system.memory.used)} / ${formatBytes(system.memory.total)}` : undefined,
    },
    {
      key: 'disk',
      percent: system?.disk.available ? system.disk.usagePercent : undefined,
      detail: system?.disk.available ? `${formatBytes(system.disk.used)} / ${formatBytes(system.disk.total)}` : undefined,
    },
  ];

  return (
    <section
      data-testid="overview-node-red-health-tile"
      className="grid gap-4 py-4 sm:py-6 lg:grid-cols-[16rem_minmax(0,1fr)] lg:gap-6"
      aria-labelledby="overview-node-red-health-title"
    >
      <header className="space-y-2">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <h2 id="overview-node-red-health-title" className="text-sm font-semibold text-ds-text-primary">
            {t('dashboard:overviewTiles.nodeRedHealth.title')}
          </h2>
          <StatusChip
            variant={overallVariant}
            size="sm"
            ariaLabel={t('dashboard:overviewTiles.nodeRedHealth.statusAria', { state: statusLabel })}
          >
            {statusLabel}
          </StatusChip>
        </div>
        <p className="text-xs text-ds-text-secondary">
          {getDeploymentLabel(host?.nodeRed.mode, t)}
          {version && (
            <>
              {' · '}
              <span className="font-mono font-semibold tabular-nums text-ds-text-primary">{`Node-RED ${version}`}</span>
            </>
          )}
        </p>
        <Link
          to="/configuration"
          className="inline-flex min-h-11 items-center gap-1 text-sm font-medium text-ds-on-tint-primary underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ds-accent-primary md:min-h-6"
          data-testid="overview-node-red-link"
        >
          {t('dashboard:overviewTiles.nodeRedHealth.cta')}
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </Link>
      </header>

      <div className="min-w-0 space-y-4">
        {!hostReady && issues.length > 0 && (
          <ul
            className="list-disc space-y-1 rounded-box border border-ds-warning/35 bg-ds-warning/10 py-2 pl-7 pr-3 text-xs text-ds-on-tint-warning"
            data-testid="overview-node-red-issues"
          >
            {issues.map((issue) => (
              <li key={issue}>{issue}</li>
            ))}
          </ul>
        )}

        <div data-testid="resource-metrics" className="space-y-2">
          <p className="text-xs text-ds-text-secondary">{scopeLabel(system?.resourceScope, t)}</p>
          <dl className="grid gap-x-6 sm:grid-cols-3" data-testid="overview-node-red-resources">
            {metrics.map((metric) => (
              <div key={metric.key} className="flex items-baseline justify-between gap-3 border-b border-ds-border-subtle py-2 sm:block">
                <dt className="text-xs text-ds-text-secondary">
                  {t(`dashboard:overviewTiles.nodeRedHealth.${metric.key}`)}
                </dt>
                <dd className="text-right sm:text-left">
                  {metric.percent === undefined ? (
                    <span className="text-sm text-ds-text-muted">{t('dashboard:overviewTiles.nodeRedHealth.notReported')}</span>
                  ) : (
                    <span className="text-sm font-semibold tabular-nums text-ds-text-primary">
                      {Math.round(metric.percent)}
                      <span className="ml-0.5 text-xs font-normal text-ds-text-secondary">%</span>
                    </span>
                  )}
                  {metric.detail && (
                    <span className="block text-xs tabular-nums text-ds-text-muted">{metric.detail}</span>
                  )}
                </dd>
              </div>
            ))}
          </dl>
        </div>

        <div
          role="group"
          aria-label={t('dashboard:overviewTiles.nodeRedHealth.actionsLabel')}
          className="flex flex-wrap items-center gap-2"
        >
          <button
            type="button"
            onClick={onOpenNodeRed}
            data-testid="overview-node-red-open"
            className="action-btn-secondary min-h-11 md:min-h-9"
          >
            <ExternalLink className="h-4 w-4" aria-hidden="true" />
            {t('dashboard:runtimeCard.openButton')}
          </button>
          <button
            type="button"
            aria-label={isRestarting
              ? t('dashboard:runtimeCard.restartingButton')
              : t('dashboard:runtimeCard.restartButton')}
            onClick={onRequestRestart}
            disabled={isRestarting || !canRestart}
            data-testid="overview-node-red-restart"
            className="action-btn-ghost min-h-11 disabled:cursor-not-allowed md:min-h-9"
          >
            <RefreshCw className={cn('h-4 w-4', isRestarting && 'animate-spin')} aria-hidden="true" />
            {isRestarting ? t('dashboard:runtimeCard.restartingButton') : t('dashboard:runtimeCard.restartButton')}
          </button>
        </div>

        {container?.image && inDocker && (
          <p className="flex min-w-0 gap-1 text-xs text-ds-text-secondary">
            <span>{t('dashboard:overviewTiles.nodeRedHealth.imageLabel')}</span>
            <span className="truncate font-mono font-semibold text-ds-text-primary" title={container.image}>
              {container.image}
            </span>
          </p>
        )}
      </div>
    </section>
  );
}
