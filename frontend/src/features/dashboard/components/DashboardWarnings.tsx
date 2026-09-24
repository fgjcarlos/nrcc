import { WarningBanner } from '@/shared/components/ui';
import type { HostStatus } from '@/shared/types';
import { getHostWarningMessage } from '../lib';
import { useT } from '@/i18n';

interface DashboardWarningsProps {
  host?: HostStatus;
  showDockerWarning: boolean;
}

export function DashboardWarnings({ host, showDockerWarning }: DashboardWarningsProps) {
  const { t } = useT();
  const hostWarningMessage =
    host && (!host.nodejs.installed || !host.nodeRed.detected || !host.settings.writable)
      ? getHostWarningMessage(host, t)
      : null;

  return (
    <>
      {showDockerWarning && (
        <WarningBanner message={t('dashboard:dockerWarning')} />
      )}

      {hostWarningMessage && <WarningBanner message={hostWarningMessage} />}
    </>
  );
}
