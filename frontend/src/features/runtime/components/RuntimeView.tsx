import { formatUptime, formatBytes, cn } from '@/shared/lib';
import { RotateCcw } from 'lucide-react';
import { useRuntimeData, useRuntimeActions } from '../hooks';

export function RuntimeView() {
  const { runtime, isLoading } = useRuntimeData();
  const { restartMutation } = useRuntimeActions();

  if (isLoading) {
    return <div className="p-4 text-base-content/70">Loading...</div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">System control</p>
        <h1 className="text-2xl font-bold text-base-content">Runtime</h1>
      </div>

      {/* Status */}
      <div className="surface-card p-6">
        <div className="flex items-center gap-4">
          <div className={cn(
            'w-4 h-4 rounded-full',
            runtime?.status === 'running' ? 'bg-green-500' : 'bg-red-500'
          )} />
          <span className="text-lg font-medium capitalize text-base-content">{runtime?.status || 'unknown'}</span>
        </div>

        {runtime?.uptime && (
          <p className="mt-4 text-3xl font-bold text-base-content">
            {formatUptime(runtime.uptime)}
          </p>
        )}
        <p className="text-base-content/60">Uptime</p>
      </div>

      {/* Actions */}
      <div className="flex gap-4">
        <button 
          onClick={() => restartMutation.mutate()}
          disabled={restartMutation.isPending}
          className="action-btn-primary"
        >
          <RotateCcw className={cn('w-4 h-4', restartMutation.isPending && 'animate-spin')} />
          {restartMutation.isPending ? 'Restarting...' : 'Restart Runtime'}
        </button>
      </div>

      {/* Info */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="surface-card p-4">
          <p className="text-sm text-base-content/60">PID</p>
          <p className="text-xl font-bold text-base-content">{runtime?.pid || '--'}</p>
        </div>
        <div className="surface-card p-4">
          <p className="text-sm text-base-content/60">Version</p>
          <p className="text-xl font-bold text-base-content">{runtime?.version || '--'}</p>
        </div>
        <div className="surface-card p-4">
          <p className="text-sm text-base-content/60">Memory RSS</p>
          <p className="text-xl font-bold text-base-content">{runtime?.memory ? formatBytes(runtime.memory.rss) : '--'}</p>
        </div>
        <div className="surface-card p-4">
          <p className="text-sm text-base-content/60">Heap Used</p>
          <p className="text-xl font-bold text-base-content">{runtime?.memory ? formatBytes(runtime.memory.heapUsed) : '--'}</p>
        </div>
      </div>
    </div>
  );
}
