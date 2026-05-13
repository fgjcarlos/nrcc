import api from '@/shared/lib';
import type { ApiResponse, SystemInfo } from '@/shared/types';

export const systemService = {
  getInfo: () => api.get<ApiResponse<SystemInfo>>('/system/info'),
};

export const healthService = {
  check: () => api.get<ApiResponse<{ status: string; uptime: number }>>('/health'),
};
