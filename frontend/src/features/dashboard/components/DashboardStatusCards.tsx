import { formatBytes } from '@/shared/lib';
import { formatPercent } from '@/features/dashboard/lib';
import type { HostStatus, SystemInfo } from '@/shared/types';
import { Cpu, MemoryStick } from 'lucide-react';
import type { DashboardContainerStatus } from '../types';
import { useSystemHistory } from '../hooks/useSystemHistory';
import { LazyMetricsChart as MetricsChart } from './LazyMetricsChart';
import { RuntimeCard } from './RuntimeCard';

interface DashboardStatusCardsProps { container?: DashboardContainerStatus | null; host?: HostStatus; inDocker: boolean; system?: SystemInfo; isRestarting: boolean; onRequestRestart: () => void; onOpenNodeRed: () => void; }
interface MetricCardProps { system?: SystemInfo; }
function CpuCard({ system }: MetricCardProps) { const { data: history, isLoading } = useSystemHistory(); return <div data-dashboard-status-card="CPU" className="p-6 border card surface-card border-border"><div className="flex items-center gap-3"><Cpu className="w-5 h-5 text-body-secondary"/><span className="text-sm font-medium">CPU</span></div><p className="mt-2 text-2xl font-bold">{system ? formatPercent(system.cpu.usage) : '--'}</p><p className="mt-1 text-sm text-body-secondary">{system?.cpu.cores || 0} cores</p><div className="mt-3"><MetricsChart data={history} dataKey="cpuPercent" label="CPU usage" color="#3b82f6" loading={isLoading}/></div></div>; }
function MemoryCard({ system }: MetricCardProps) { const { data: history, isLoading } = useSystemHistory(); return <div data-dashboard-status-card="Memory" className="p-6 border card surface-card border-border"><div className="flex items-center gap-3"><MemoryStick className="w-5 h-5 text-body-secondary"/><span className="text-sm font-medium">Memory</span></div><p className="mt-2 text-2xl font-bold">{system ? formatPercent(system.memory.usagePercent) : '--'}</p><p className="mt-1 text-sm text-body-secondary">{system ? `${formatBytes(system.memory.used)} / ${formatBytes(system.memory.total)}` : '--'}</p><div className="mt-3"><MetricsChart data={history} dataKey="memoryPercent" label="Memory usage" color="#8b5cf6" loading={isLoading}/></div></div>; }
export function DashboardStatusCards(props: DashboardStatusCardsProps) { return <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3"><RuntimeCard container={props.container} host={props.host} inDocker={props.inDocker} isRestarting={props.isRestarting} onRequestRestart={props.onRequestRestart} onOpenNodeRed={props.onOpenNodeRed}/><CpuCard system={props.system}/><MemoryCard system={props.system}/></div>; }
