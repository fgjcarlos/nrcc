import { logService } from '@/features/logs/services';
import type { LogEntry, LogLevel } from '@/shared/types';

const toTxt = (logs: LogEntry[]) =>
  logs.map(l => `${l.timestamp} [${l.level.toUpperCase()}] ${l.message}`).join('\n');

const ts = () => new Date().toISOString().replace(/[:.]/g, '-');

const triggerDownload = (blob: Blob, filename: string) => {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
};

export function useLogsActions() {
  const handleClear = async (refetch: () => void) => {
    await logService.clearLogs();
    refetch();
  };

  const handleDownload = (logs: LogEntry[]) => {
    triggerDownload(new Blob([toTxt(logs)], { type: 'text/plain' }), `nodered-logs-${ts()}.txt`);
  };

  const handleDownloadJSON = (logs: LogEntry[]) => {
    triggerDownload(
      new Blob([JSON.stringify(logs, null, 2)], { type: 'application/json' }),
      `nodered-logs-${ts()}.json`,
    );
  };

  const handleCopy = async (logs: LogEntry[]) => {
    if (!navigator.clipboard?.writeText) return;
    await navigator.clipboard.writeText(toTxt(logs));
  };

  const toggleLevel = (current: LogLevel[], level: LogLevel): LogLevel[] =>
    current.includes(level) ? current.filter(l => l !== level) : [...current, level];

  const getLevelColor = (level: LogLevel) => {
    switch (level) {
      case 'error':
        return 'text-red-500';
      case 'warn':
        return 'text-yellow-500';
      case 'debug':
        return 'text-gray-500';
      default:
        return 'text-blue-500';
    }
  };

  return {
    handleClear,
    handleCopy,
    handleDownload,
    handleDownloadJSON,
    toggleLevel,
    getLevelColor,
  };
}
