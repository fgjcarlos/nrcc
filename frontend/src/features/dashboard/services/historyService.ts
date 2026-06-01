import api from '@/shared/lib';
import type { ApiResponse } from '@/shared/types';
import type { RuntimeInfo } from '@/shared/types';
import type { MetricsSnapshot, RestartEvent } from '../types/history';

export interface SystemHistoryResponse {
  data: MetricsSnapshot[];
  timestamp: string;
}

export interface RuntimeHistoryResponse {
  data: {
    events: RestartEvent[];
    status: RuntimeInfo;
  };
  timestamp: string;
}

export const historyService = {
  getSystemHistory: (n = 120) =>
    api.get<ApiResponse<MetricsSnapshot[]>>('/system/history', { params: { n } }),

  getRuntimeHistory: (n = 50) =>
    api.get<ApiResponse<{ events: RestartEvent[]; status: RuntimeInfo }>>('/runtime/history', { params: { n } }),
};
