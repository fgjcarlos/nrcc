import { cn } from '@/shared/lib';
import { StateContainer } from '@/shared/components';
import { Trash2, Download, Play, Pause, Copy, FileJson, RefreshCw } from 'lucide-react';
import type { LogEntry, LogLevel } from '@/shared/types';
import { useLogsData, useLogsActions } from '../hooks';

const LEVELS: LogLevel[] = ['debug', 'info', 'warn', 'error'];

export function LogsView() {
  const { logs, isLoading, isError, levelFilter, setLevelFilter, isPaused, setIsPaused, refetch } = useLogsData();
  const { handleClear, handleCopy, handleDownload, handleDownloadJSON, toggleLevel, getLevelColor } = useLogsActions();

  return (
    <div className="flex h-full flex-col space-y-6 p-6">
      <div>
        <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Observability</p>
        <h1 className="text-3xl font-bold tracking-tight text-base-content">Logs</h1>
        <p className="mt-2 max-w-2xl text-sm text-base-content/65">
          Stream Node-RED runtime logs, filter by level and export a snapshot when needed.
        </p>
      </div>

      <div className="surface-panel flex flex-col gap-3 border border-border p-3 md:flex-row md:flex-wrap md:items-center">
        {/* Level chips */}
        <div role="group" aria-label="Filter logs by level" className="flex flex-wrap items-center gap-2">
          {LEVELS.map(level => {
            const active = levelFilter.includes(level);
            return (
              <button
                key={level}
                type="button"
                role="switch"
                aria-checked={active}
                aria-label={`Toggle ${level} level`}
                onClick={() => setLevelFilter(toggleLevel(levelFilter, level))}
                className={cn(
                  'rounded-full border border-border px-3 py-1 text-xs font-medium uppercase tracking-wide transition-colors',
                  active
                    ? 'bg-primary/15 text-primary border-primary/40'
                    : 'bg-base-300/40 text-base-content/55 hover:bg-base-300/70',
                )}
              >
                {level}
              </button>
            );
          })}
        </div>

        {/* Action buttons. Order matters for LogsView.test.tsx — first three
            buttons map to Pause, Clear, Download respectively. */}
        <div className="flex flex-wrap items-center gap-2 md:ml-auto">
          <IconButton
            onClick={() => setIsPaused(!isPaused)}
            label={isPaused ? 'Resume' : 'Pause'}
            icon={isPaused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
            tone={isPaused ? 'warning' : 'default'}
          />
          <IconButton
            onClick={() => handleClear(refetch)}
            label="Clear"
            icon={<Trash2 className="h-4 w-4" />}
          />
          <IconButton
            onClick={() => handleDownload(logs)}
            label=".txt"
            icon={<Download className="h-4 w-4" />}
          />
          <IconButton
            onClick={() => handleDownloadJSON(logs)}
            label=".json"
            icon={<FileJson className="h-4 w-4" />}
          />
          <IconButton
            onClick={() => handleCopy(logs)}
            label="Copy"
            icon={<Copy className="h-4 w-4" />}
          />
          <IconButton
            onClick={() => refetch()}
            label="Refresh"
            icon={<RefreshCw className="h-4 w-4" />}
          />
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
            {logs.map((log: LogEntry) => (
              <div key={log.id} className="flex gap-2 rounded-xl px-2 py-2 transition-colors hover:bg-base-300/50">
                <span className="shrink-0 text-base-content/55">
                  {new Date(log.timestamp).toLocaleTimeString()}
                </span>
                <span className={cn('w-14 shrink-0 font-bold', getLevelColor(log.level))}>
                  {log.level.toUpperCase()}
                </span>
                <span className="whitespace-pre-wrap break-words text-base-content">{log.message}</span>
              </div>
            ))}
          </div>
        </StateContainer>
      </div>
    </div>
  );
}

type IconButtonProps = {
  onClick: () => void;
  label: string;
  icon: React.ReactNode;
  tone?: 'default' | 'warning';
};

function IconButton({ onClick, label, icon, tone = 'default' }: IconButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'inline-flex items-center gap-2 rounded-xl border border-border px-3 py-2 text-sm transition-colors',
        tone === 'warning'
          ? 'bg-warning/10 text-warning hover:bg-warning/20'
          : 'bg-base-300/60 text-base-content hover:bg-base-300/80',
      )}
    >
      {icon}
      <span>{label}</span>
    </button>
  );
}
