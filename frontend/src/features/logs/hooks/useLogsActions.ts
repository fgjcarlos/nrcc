import { logService } from '@/features/logs/services';
import type { LogLevel } from '@/shared/types';

export function useLogsActions() {
  const handleClear = async (refetch: () => void) => {
    await logService.clearLogs();
    refetch();
  };

  const handleDownload = (logs: any[]) => {
    const content = logs.map(l => `${l.timestamp} [${l.level.toUpperCase()}] ${l.message}`).join('\n');
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `nodered-logs-${new Date().toISOString()}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

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
    handleDownload,
    getLevelColor,
  };
}
