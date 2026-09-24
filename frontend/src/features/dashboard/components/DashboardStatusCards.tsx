import { formatBytes } from '@/shared/lib';
import { formatPercent } from '@/features/dashboard/lib';
import type { HostStatus, RuntimeInfo, SystemInfo } from '@/shared/types';
import { Cpu, ExternalLink, MemoryStick, RefreshCw, Server } from 'lucide-react';
import type { DashboardContainerStatus } from '../types';
import { getDeploymentLabel } from '../lib';
import { useSystemHistory } from '../hooks/useSystemHistory';
import { LazyMetricsChart as MetricsChart } from './LazyMetricsChart';
import { cn } from '@/shared/lib';
import { useT } from '@/i18n';

interface DashboardStatusCardsProps {
  container?: DashboardContainerStatus | null;
  host?: HostStatus;
  runtime?: RuntimeInfo;
  inDocker: boolean;
  system?: SystemInfo;
  isRestarting: boolean;
  onRequestRestart: () => void;
  onOpenNodeRed: () => void;
}

// "Runtime" card — promoted out of the metric row (issue #676 item 1).
// Carries Node-RED process state, container image / settings path, and
// the Restart + Open actions that previously lived further down the page
// in QuickActionsCard (DashboardDetails.tsx).
function RuntimeCard({
  container,
  host,
  runtime,
  inDocker,
  isRestarting,
  onRequestRestart,
  onOpenNodeRed,
}: Pick<
  DashboardStatusCardsProps,
  'container' | 'host' | 'runtime' | 'inDocker' | 'isRestarting' | 'onRequestRestart' | 'onOpenNodeRed'
>) {
  const { t } = useT();
  const runtimeStatus = runtime?.status ?? (host?.nodeRed.running ? 'running' : 'unknown');

  // canRestart is true only when the runtime is actually present. The
  // button stays mounted regardless so the user always sees the action
  // affordance; only the enabled state changes. Without this gate the
  // fallback telemetry card would either hide the action row or let
  // through a restart that has nothing to act on.
  const canRestart = Boolean(host?.nodeRed?.detected);
  const restartLabel = isRestarting
    ? t('dashboard:runtimeCard.restartingButton')
    : t('dashboard:runtimeCard.restartButton');
  const openLabel = t('dashboard:runtimeCard.openButton');

  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        <Server className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-medium">{t('dashboard:runtimeCard.title')}</span>
      </div>
      {inDocker ? (
        <>
          <p className="mt-2 text-2xl font-bold capitalize">{runtimeStatus}</p>
          <p className="mt-1 text-sm truncate text-body-secondary" title={container?.image}>
            {runtime?.pid ? `PID ${runtime.pid} · ${container?.image || 'Docker'}` : container?.image || 'Docker'}
          </p>
        </>
      ) : (
        <>
          <p className="mt-2 text-2xl font-bold">{getDeploymentLabel(host?.nodeRed.mode, t)}</p>
          <p className="mt-1 text-sm text-body-secondary">{host?.settings.path || 'No settings.js path detected'}</p>
        </>
      )}

      {/* Actions co-located with the process state they act on (issue #676 item 1).
          Always mounted so the action affordance survives any telemetry
          fallback; only the enabled state changes based on availability. */}
      <div className="grid grid-cols-2 gap-2.5 mt-5">
        <button
          type="button"
          aria-label={restartLabel}
          onClick={onRequestRestart}
          disabled={isRestarting || !canRestart}
          className="group action-btn-secondary flex items-center justify-center gap-3 rounded-xl p-4"
        >
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-warning/10 text-warning transition-colors group-hover:bg-warning/20">
            <RefreshCw className={cn('w-4 h-4', isRestarting && 'animate-spin')} />
          </div>
          <span className="text-base font-medium">{restartLabel}</span>
        </button>
        <button
          type="button"
          aria-label={openLabel}
          onClick={onOpenNodeRed}
          className="group action-btn-secondary flex items-center justify-center gap-3 rounded-xl p-4"
        >
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-info/10 text-info transition-colors group-hover:bg-info/20">
            <ExternalLink className="w-4 h-4" />
          </div>
          <span className="text-base font-medium">{openLabel}</span>
        </button>
      </div>
    </div>
  );
}

interface MetricCardProps {
  system?: SystemInfo;
}

function scopeLabel(scope?: SystemInfo['resourceScope']) {
  if (scope === 'container') return 'Container scope';
  if (scope === 'host') return 'Host scope';
  return 'Resource metrics unavailable';
}

function CpuCard({ system }: MetricCardProps) {
  const { data: history, isLoading } = useSystemHistory();

  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        <Cpu className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-medium">CPU</span>
      </div>
      <p className="mt-2 text-2xl font-bold">{system?.cpu.available ? formatPercent(system.cpu.usage) : 'Unavailable'}</p>
      <p className="mt-1 text-sm text-body-secondary">{system?.cpu.available ? `${system.cpu.cores} cores` : scopeLabel(system?.resourceScope)}</p>
      <div className="mt-3">
        <MetricsChart
          data={history}
          dataKey="cpuPercent"
          label="CPU usage"
          color="var(--color-accent)"
          loading={isLoading}
        />
      </div>
    </div>
  );
}

function MemoryCard({ system }: MetricCardProps) {
  const { data: history, isLoading } = useSystemHistory();

  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        <MemoryStick className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-medium">Memory</span>
      </div>
      <p className="mt-2 text-2xl font-bold">{system?.memory.available ? formatPercent(system.memory.usagePercent) : 'Unavailable'}</p>
      <p className="mt-1 text-sm text-body-secondary">
        {system?.memory.available ? `${formatBytes(system.memory.used)} / ${formatBytes(system.memory.total)}` : scopeLabel(system?.resourceScope)}
      </p>
      <div className="mt-3">
        <MetricsChart
          data={history}
          dataKey="memoryPercent"
          label="Memory usage"
          color="var(--color-info)"
          loading={isLoading}
        />
      </div>
    </div>
  );
}

// Disk moved out of the top metric row (issue #676 item 3). The detail
// breakdown now lives in DiskUsageCard inside DashboardDetails.tsx.

export function DashboardStatusCards(props: DashboardStatusCardsProps) {
  return (
    <section aria-label="Resource metrics" data-testid="resource-metrics" className="space-y-2">
      <p className="text-xs font-medium uppercase tracking-wide text-body-secondary">{scopeLabel(props.system?.resourceScope)}</p>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <RuntimeCard
        container={props.container}
        host={props.host}
        runtime={props.runtime}
        inDocker={props.inDocker}
        isRestarting={props.isRestarting}
        onRequestRestart={props.onRequestRestart}
        onOpenNodeRed={props.onOpenNodeRed}
      />
      <CpuCard system={props.system} />
      <MemoryCard system={props.system} />
      </div>
    </section>
  );
}
