import { formatBytes, formatUptime, cn } from '@/shared/lib';
import { formatPercent } from '@/features/dashboard/lib';
import type { HostStatus, RuntimeInfo, SystemInfo } from '@/shared/types';
import { Cpu, MemoryStick, RefreshCw, Server } from 'lucide-react';
import type { DashboardContainerStatus } from '../types';
import { getDeploymentLabel, getStatusBadgeClass } from '../lib';

interface DashboardStatusCardsProps {
  container?: DashboardContainerStatus | null;
  host?: HostStatus;
  inDocker: boolean;
  isRestarting: boolean;
  runtime?: RuntimeInfo;
  runtimeStatus: string;
  system?: SystemInfo;
}

function RuntimeStatusCard({
  isRestarting,
  runtime,    
  runtimeStatus,
}: Pick<
  DashboardStatusCardsProps,
  'isRestarting' | 'runtime' | 'runtimeStatus'
>) {
  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        {isRestarting ? (
          <RefreshCw className="w-5 h-5 animate-spin text-warning" />
        ) : (
          <span className={cn('badge badge-lg', getStatusBadgeClass(runtime?.status || 'unknown'))} />
        )}
        <span className="text-sm font-medium">Node-RED</span>
      </div>
      <p className={cn('mt-2 text-2xl font-bold capitalize', isRestarting && 'text-warning')}>{runtimeStatus}</p>
      {isRestarting ? (
        <p className="mt-1 text-sm text-body-secondary">Espera un momento…</p>
      ) : runtime?.uptime ? (
        <p className="mt-1 text-sm text-body-secondary">Uptime: {formatUptime(runtime.uptime)}</p>
      ) : (
        <p className="mt-1 text-sm text-body-secondary">—</p>
      )}
    </div>
  );
}

function DeploymentCard({ container, host, inDocker }: Pick<DashboardStatusCardsProps, 'container' | 'host' | 'inDocker'>) {
  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        <Server className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-medium">Deployment</span>
      </div>
      {inDocker ? (
        <>
          <p className="mt-2 text-2xl font-bold capitalize">{container?.status ?? '—'}</p>
          <p className="mt-1 text-sm truncate text-body-secondary" title={container?.image}>
            {container?.image || 'Docker'}
          </p>
        </>
      ) : (
        <>
          <p className="mt-2 text-2xl font-bold">{getDeploymentLabel(host?.nodeRed.mode)}</p>
          <p className="mt-1 text-sm text-body-secondary">{host?.settings.path || 'Sin ruta de settings.js detectada'}</p>
        </>
      )}
    </div>
  );
}

function CpuCard({ system }: Pick<DashboardStatusCardsProps, 'system'>) {
  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        <Cpu className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-medium">CPU</span>
      </div>
      <p className="mt-2 text-2xl font-bold">{system ? formatPercent(system.cpu.usage) : '--'}</p>
      <p className="mt-1 text-sm text-body-secondary">{system?.cpu.cores || 0} cores</p>
    </div>
  );
}

function MemoryCard({ system }: Pick<DashboardStatusCardsProps, 'system'>) {
  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        <MemoryStick className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-medium">Memory</span>
      </div>
      <p className="mt-2 text-2xl font-bold">{system ? formatPercent(system.memory.usagePercent) : '--'}</p>
      <p className="mt-1 text-sm text-body-secondary">
        {system ? `${formatBytes(system.memory.used)} / ${formatBytes(system.memory.total)}` : '--'}
      </p>
    </div>
  );
}

export function DashboardStatusCards(props: DashboardStatusCardsProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
      <RuntimeStatusCard {...props} />
      <DeploymentCard container={props.container} host={props.host} inDocker={props.inDocker} />
      <CpuCard system={props.system} />
      <MemoryCard system={props.system} />
    </div>
  );
}
