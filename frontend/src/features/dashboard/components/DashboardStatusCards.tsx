import { formatBytes } from '@/shared/lib';
import { formatPercent } from '@/features/dashboard/lib';
import type { HostStatus, SystemInfo } from '@/shared/types';
import { Cpu, HardDrive, MemoryStick, Server } from 'lucide-react';
import type { DashboardContainerStatus } from '../types';
import { getDeploymentLabel } from '../lib';
import { useSystemHistory } from '../hooks/useSystemHistory';
import { MetricsChart } from './MetricsChart';

interface DashboardStatusCardsProps {
  container?: DashboardContainerStatus | null;
  host?: HostStatus;
  inDocker: boolean;
  system?: SystemInfo;
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

interface MetricCardProps {
  system?: SystemInfo;
}

function CpuCard({ system }: MetricCardProps) {
  const { data: history, isLoading } = useSystemHistory();

  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        <Cpu className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-medium">CPU</span>
      </div>
      <p className="mt-2 text-2xl font-bold">{system ? formatPercent(system.cpu.usage) : '--'}</p>
      <p className="mt-1 text-sm text-body-secondary">{system?.cpu.cores || 0} cores</p>
      <div className="mt-3">
        <MetricsChart
          data={history}
          dataKey="cpuPercent"
          label="CPU usage"
          color="#3b82f6"
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
      <p className="mt-2 text-2xl font-bold">{system ? formatPercent(system.memory.usagePercent) : '--'}</p>
      <p className="mt-1 text-sm text-body-secondary">
        {system ? `${formatBytes(system.memory.used)} / ${formatBytes(system.memory.total)}` : '--'}
      </p>
      <div className="mt-3">
        <MetricsChart
          data={history}
          dataKey="memoryPercent"
          label="Memory usage"
          color="#8b5cf6"
          loading={isLoading}
        />
      </div>
    </div>
  );
}

function DiskCard({ system }: MetricCardProps) {
  const { data: history, isLoading } = useSystemHistory();

  return (
    <div className="p-6 border card surface-card border-border">
      <div className="flex items-center gap-3">
        <HardDrive className="w-5 h-5 text-body-secondary" />
        <span className="text-sm font-medium">Disk</span>
      </div>
      <p className="mt-2 text-2xl font-bold">{system ? formatPercent(system.disk.usagePercent) : '--'}</p>
      <p className="mt-1 text-sm text-body-secondary">
        {system ? `${formatBytes(system.disk.used)} / ${formatBytes(system.disk.total)}` : '--'}
      </p>
      <div className="mt-3">
        <MetricsChart
          data={history}
          dataKey="diskPercent"
          label="Disk usage"
          color="#10b981"
          loading={isLoading}
        />
      </div>
    </div>
  );
}

export function DashboardStatusCards(props: DashboardStatusCardsProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      <DeploymentCard container={props.container} host={props.host} inDocker={props.inDocker} />
      <CpuCard system={props.system} />
      <MemoryCard system={props.system} />
      <DiskCard system={props.system} />
    </div>
  );
}
