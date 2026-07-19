import api from '@/shared/lib';
import type { ApiResponse } from '@/shared/types';
import { bootstrapService } from '@/features/bootstrap/services';
import type { ContainerInfo } from '@/shared/types';
import { systemService } from './systemService';
import { configService } from '@/features/configuration/services';
import { backupService } from '@/features/backups/services';

export const dashboardService = {
  // Lightweight read of the container status. The /docker page itself was
  // removed in #477; this is the only field the dashboard's status card
  // still needs. The endpoint is kept because the handler answers with a
  // synthetic record when running inside a container, so the operator
  // always sees a usable response without having to wire `docker-cli`.
  getDockerStatus: () => api.get<ApiResponse<ContainerInfo>>('/docker/status'),
  getSystemInfo: () => systemService.getInfo(),
  getConfig: () => configService.getConfig(),
  getHostStatus: () => bootstrapService.getStatus(),
  getBackupObservability: () => backupService.getObservability(),
  restartNodeRed: () => api.post<ApiResponse<{ message: string }>>('/runtime/restart'),
  startNodeRed: () => api.post<ApiResponse<{ message: string }>>('/runtime/start'),
  stopNodeRed: () => api.post<ApiResponse<{ message: string }>>('/runtime/stop'),
};
