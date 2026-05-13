import { bootstrapService } from '@/features/bootstrap/services';
import { dockerService } from '@/features/docker/services';
import { systemService } from './systemService';
import { configService, runtimeService } from '@/features/configuration/services';
import { backupService } from '@/features/backups/services';

export const dashboardService = {
  getRuntimeStatus: () => runtimeService.getStatus(),
  getDockerStatus: () => dockerService.getStatus(),
  getSystemInfo: () => systemService.getInfo(),
  getConfig: () => configService.getConfig(),
  getHostStatus: () => bootstrapService.getStatus(),
  getBackupObservability: () => backupService.getObservability(),
  restartNodeRed: () => runtimeService.restart(),
  startNodeRed: () => runtimeService.start(),
  stopNodeRed: () => runtimeService.stop(),
};
