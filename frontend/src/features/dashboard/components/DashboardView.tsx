import { useDashboardActions, useDashboardData } from '../hooks';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { DashboardDetails } from './DashboardDetails';
import { DashboardHeader } from './DashboardHeader';
import { DashboardStatusCards } from './DashboardStatusCards';
import { DashboardWarnings } from './DashboardWarnings';
import { RestartConfirmationModal } from './RestartConfirmationModal';
import { SystemHealthCard } from './SystemHealthCard';
import { SecurityPostureCard } from './SecurityPostureCard';

export function DashboardView() {
  const { container, system, config, host, backups, securityPosture, securityPostureLoading, securityPostureError, dockerSuccess, dockerLoading, dockerError } = useDashboardData();
  const { user } = useAuth();
  const {
    pendingConfirm,
    isRestarting,
    setPendingConfirm,
    handleRestartConfirm,
    handleOpenNodeRed,
  } = useDashboardActions({ uiPort: config?.uiPort as number | undefined });

  const inDocker = !!container?.inDocker;
  const showDockerWarning =
    !dockerLoading &&
    !dockerError &&
    dockerSuccess &&
    !!container?.status &&
    container.status !== 'running';

  return (
    <div className="space-y-8">
      <DashboardHeader edgeMode={system?.edgeMode} />
      <DashboardWarnings showDockerWarning={showDockerWarning} host={host} />
      <SystemHealthCard host={host} />
      <DashboardStatusCards
        system={system}
        host={host}
        inDocker={inDocker}
        container={container}
        isRestarting={isRestarting}
        onRequestRestart={() => setPendingConfirm(true)}
        onOpenNodeRed={handleOpenNodeRed}
      />
      {user?.role === 'admin' ? <SecurityPostureCard posture={securityPosture} isLoading={securityPostureLoading} isError={securityPostureError} /> : null}
      <DashboardDetails
        system={system}
        backups={backups}
      />
      <RestartConfirmationModal
        isOpen={pendingConfirm}
        onConfirm={handleRestartConfirm}
        onCancel={() => setPendingConfirm(false)}
      />
    </div>
  );
}
