import { WarningBanner } from '@/shared/components/ui';
import type { HostStatus } from '@/shared/types';
import { getHostWarningMessage } from '../lib';

interface DashboardWarningsProps {
  host?: HostStatus;
  showDockerWarning: boolean;
}

export function DashboardWarnings({ host, showDockerWarning }: DashboardWarningsProps) {
  const hostWarningMessage =
    host && (!host.nodejs.installed || !host.nodeRed.detected || !host.settings.writable)
      ? getHostWarningMessage(host)
      : null;

  return (
    <>
      {showDockerWarning && (
        <WarningBanner message="Docker container is not running correctly. Some features may not work." />
      )}

      {hostWarningMessage && <WarningBanner message={hostWarningMessage} />}
    </>
  );
}
