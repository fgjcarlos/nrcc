import { cn } from '@/shared/lib';
import { StateContainer } from '@/shared/components';
import { Trash2, Download, Play, Pause } from 'lucide-react';
import { useLogsData, useLogsActions } from '../hooks';

export function LogsView() {
  const { logs, isLoading, isError, levelFilter, setLevelFilter, isPaused, setIsPaused, refetch } = useLogsData();
  const { handleClear, handleDownload, getLevelColor } = useLogsActions();

  return (
    <div className="flex h-full flex-col space-y-6 p-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Observability</p>
          <h1 className="text-3xl font-bold tracking-tight text-base-content">Logs</h1>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Stream Node-RED runtime logs, filter by level and export a snapshot when needed.
          </p>
        </div>
        
        <div className="surface-panel flex flex-wrap items-center gap-2 border border-border p-3">
          <select
            multiple
            value={levelFilter}
            onChange={(e) => setLevelFilter(Array.from(e.target.selectedOptions, o => o.value as LogLevel))}
            className="min-h-11 rounded-xl border border-border bg-base-100/70 px-3 py-2 text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary"
          >
            <option value="debug">Debug</option>
            <option value="info">Info</option>
            <option value="warn">Warn</option>
            <option value="error">Error</option>
          </select>

          <button
            onClick={() => setIsPaused(!isPaused)}
            className={cn(
              'rounded-xl border border-border p-3 transition-colors',
              isPaused ? 'bg-warning/10 text-warning' : 'bg-base-300/60 text-base-content hover:bg-base-300/80'
            )}
          >
            {isPaused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
          </button>

          <button
            onClick={() => handleClear(refetch)}
            className="rounded-xl border border-border p-3 text-base-content transition-colors hover:bg-base-300/60"
          >
            <Trash2 className="w-4 h-4" />
          </button>

          <button
            onClick={() => handleDownload(logs)}
            className="rounded-xl border border-border p-3 text-base-content transition-colors hover:bg-base-300/60"
          >
            <Download className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Log Viewer */}
      <div className="surface-card flex-1 overflow-auto border border-border p-4 font-mono text-sm">
        <StateContainer
          isLoading={isLoading}
          isError={isError}
          isEmpty={logs.length === 0}
          emptySlot={
            <div className="flex h-full min-h-48 items-center justify-center rounded-2xl border border-dashed border-border py-8 text-center text-base-content/60">
              No logs available
            </div>
          }
        >
          <div className="space-y-1">
            {logs.map((log) => (
              <div key={log.id} className="flex gap-2 rounded-xl px-2 py-2 transition-colors hover:bg-base-300/50">
                <span className="shrink-0 text-base-content/55">
                  {new Date(log.timestamp).toLocaleTimeString()}
                </span>
                <span className={cn('font-bold shrink-0 w-12', getLevelColor(log.level))}>
                  {log.level.toUpperCase()}
                </span>
                <span className="text-base-content">{log.message}</span>
              </div>
            ))}
          </div>
        </StateContainer>
      </div>
    </div>
  );
}
