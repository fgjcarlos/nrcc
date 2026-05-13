import api from '@/shared/lib';
import type { ApiResponse, HostStatus } from '@/shared/types';

export const bootstrapService = {
  getStatus: () => api.get<ApiResponse<HostStatus>>('/bootstrap/status'),
};
