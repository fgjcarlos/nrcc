import api from '@/shared/lib';
import type { ApiResponse, ContainerInfo, DockerInfo } from '@/shared/types';

export const dockerService = {
  getStatus: () => api.get<ApiResponse<ContainerInfo>>('/docker/status'),
  
  getInfo: () => api.get<ApiResponse<DockerInfo>>('/docker/info'),
  
  restart: () => api.post<ApiResponse<{ message: string }>>('/docker/restart'),
  
  stop: () => api.post<ApiResponse<{ message: string }>>('/docker/stop'),
};
