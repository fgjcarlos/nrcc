import { useDashboardActions, useDashboardData } from '../hooks';
import { DashboardDetails } from './DashboardDetails';
import { DashboardHeader } from './DashboardHeader';
import { DashboardStatusCards } from './DashboardStatusCards';
import { DashboardWarnings } from './DashboardWarnings';
import { RestartConfirmationModal } from './RestartConfirmationModal';
import { SystemHealthCard } from './SystemHealthCard';

export function DashboardView() {
  const { container, system, config, host, backups, dockerSuccess, dockerLoading, dockerError } = useDashboardData();
  const {
    pendingConfirm,
    isRestarting,
    setPendingConfirm,
    handleRestartConfirm,
    handleOpenNodeRed,
  } = useDashboardActions({ uiPort: config?.uiPort });

  const inDocker = !!container?.inDocker;
  const showDockerWarning =
    !dockerLoading &&
    !dockerError &&
    dockerSuccess &&
    !!container?.status &&
    container.status !== 'running';

  return (
    <div className="space-y-8">
      <DashboardHeader />
      <DashboardWarnings showDockerWarning={showDockerWarning} host={host} />
      <SystemHealthCard host={host} />
      <DashboardStatusCards
        system={system}
        host={host}
        inDocker={inDocker}
        container={container}
      />
      <DashboardDetails
        system={system}
        backups={backups}
        isRestarting={isRestarting}
        onRequestRestart={() => setPendingConfirm(true)}
        onOpenNodeRed={handleOpenNodeRed}
      />
      <RestartConfirmationModal
        isOpen={pendingConfirm}
        onConfirm={handleRestartConfirm}
        onCancel={() => setPendingConfirm(false)}
      />
    </div>
  );
}
