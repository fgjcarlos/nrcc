import { ExternalLink, RefreshCw, Server } from 'lucide-react';
import { cn } from '@/shared/lib';
import type { HostStatus } from '@/shared/types';
import type { DashboardContainerStatus } from '../types';

export interface RuntimeCardProps {
  container?: DashboardContainerStatus | null;
  host?: HostStatus;
  inDocker: boolean;
  isRestarting: boolean;
  onRequestRestart: () => void;
  onOpenNodeRed: () => void;
}

function runtimePresentation({ container, host, inDocker }: Pick<RuntimeCardProps, 'container' | 'host' | 'inDocker'>) {
  if (inDocker) {
    return { status: container?.status ?? 'unavailable', detail: container?.image ?? 'unavailable' };
  }
  return { status: host?.nodeRed.mode ?? 'unavailable', detail: host?.settings?.path ?? 'native' };
}

function activateOnKeyboard(event: React.KeyboardEvent<HTMLButtonElement>, action: () => void) {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault();
    action();
  }
}

export function RuntimeCard(props: RuntimeCardProps) {
  const presentation = runtimePresentation(props);
  const statusTone = presentation.status === 'running' ? 'text-success' : presentation.status === 'unavailable' ? 'text-error' : 'text-warning';

  return (
    <section data-dashboard-status-card="Runtime" className="p-6 border card surface-card border-border" aria-label="Runtime">
      <div className="flex items-center gap-3"><Server className="w-5 h-5 text-body-secondary" /><span className="text-sm font-medium">Runtime</span></div>
      <p className={cn('mt-2 text-2xl font-bold capitalize', statusTone)}>{presentation.status}</p>
      <p className="mt-1 text-sm truncate text-body-secondary" title={presentation.detail}>{presentation.detail}</p>
      <div className="grid grid-cols-2 gap-2 mt-4">
        <button type="button" disabled={props.isRestarting} onClick={props.onRequestRestart} onKeyDown={(event) => activateOnKeyboard(event, props.onRequestRestart)} className="action-btn-secondary rounded-xl p-3 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary">
          <RefreshCw className={cn('inline w-4 h-4 mr-2', props.isRestarting && 'animate-spin')} />{props.isRestarting ? 'Restarting…' : 'Restart'}
        </button>
        <button type="button" onClick={props.onOpenNodeRed} onKeyDown={(event) => activateOnKeyboard(event, props.onOpenNodeRed)} className="action-btn-secondary rounded-xl p-3 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary">
          <ExternalLink className="inline w-4 h-4 mr-2" />Open
        </button>
      </div>
    </section>
  );
}
