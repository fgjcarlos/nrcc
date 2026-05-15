// Re-export dashboard-specific types from shared
// These were originally in shared/types but are only used by dashboard
export type { SystemInfo } from '@/shared/types';
import type { HostStatus } from '@/shared/types';
import type { BackupObservability } from '@/features/backups/services';

export interface DashboardContainerStatus {
  inDocker: boolean;
  status: string;
  image?: string;
}

export interface DashboardData {
  container?: DashboardContainerStatus | null;
  system?: SystemInfo;
  config?: Record<string, unknown>;
  host?: HostStatus;
  backups?: BackupObservability;
  dockerSuccess: boolean;
  dockerLoading: boolean;
  dockerError: boolean;
}
