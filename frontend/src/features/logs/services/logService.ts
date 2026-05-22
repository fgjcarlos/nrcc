import api from '@/shared/lib';
import type { ApiResponse, LogEntry } from '@/shared/types';

export const logService = {
  getLogs: (tail: number = 100, level?: string, search?: string) => {
    const params = new URLSearchParams();
    params.append('tail', tail.toString());
    if (level) params.append('level', level);
    if (search) params.append('search', search);
    return api.get<ApiResponse<LogEntry[]>>(`/runtime/logs?${params}`);
  },

  clearLogs: () => api.delete<ApiResponse<{ message: string }>>('/runtime/logs'),
  
  streamLogs: () => {
    return new EventSource(`${api.defaults.baseURL}/runtime/logs/stream`);
  },
};
