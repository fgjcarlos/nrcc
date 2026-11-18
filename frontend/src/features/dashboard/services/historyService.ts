import api from '@/shared/lib';
import type { ApiResponse } from '@/shared/types';
import type { RuntimeInfo } from '@/shared/types';
import type { RestartEvent } from '../types/history';

export interface RuntimeHistoryResponse {
  data: {
    events: RestartEvent[];
    status: RuntimeInfo;
  };
  timestamp: string;
}

export const historyService = {
  getRuntimeHistory: (n = 50) =>
    api.get<ApiResponse<{ events: RestartEvent[]; status: RuntimeInfo }>>('/runtime/history', { params: { n } }),
};
