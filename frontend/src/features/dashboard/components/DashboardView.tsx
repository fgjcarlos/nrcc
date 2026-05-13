import { useDashboardActions, useDashboardData } from '../hooks';
import { DashboardDetails } from './DashboardDetails';
import { DashboardHeader } from './DashboardHeader';
import { DashboardStatusCards } from './DashboardStatusCards';
import { DashboardWarnings } from './DashboardWarnings';
import { RestartConfirmationModal } from './RestartConfirmationModal';
import { SystemHealthCard } from './SystemHealthCard';

export function DashboardView() {
  const { runtime, container, system, config, host, backups, dockerSuccess, dockerLoading, dockerError } = useDashboardData();
  const {
    pendingConfirm,
    isRestarting,
    isStartStopping,
    setPendingConfirm,
    handleRestartConfirm,
    handleStartNodeRed,
    handleStopNodeRed,
    handleOpenNodeRed,
  } = useDashboardActions({ uiPort: config?.uiPort });

  const inDocker = !!container?.inDocker;
  const showDockerWarning =
    !dockerLoading &&
    !dockerError &&
    dockerSuccess &&
    !!container?.status &&
    container.status !== 'running';
  const runtimeStatus = isRestarting ? 'reiniciando' : runtime?.status || 'unknown';

  return (
    <div className="space-y-8">
      <DashboardHeader />
      <DashboardWarnings showDockerWarning={showDockerWarning} host={host} />
      <SystemHealthCard host={host} />
      <DashboardStatusCards
        runtimeStatus={runtimeStatus}
        isRestarting={isRestarting}
        runtime={runtime}
        system={system}
        host={host}
        inDocker={inDocker}
        container={container}
      />
      <DashboardDetails
        system={system}
        backups={backups}
        isStartStopping={isStartStopping}
        onStartNodeRed={handleStartNodeRed}
        onStopNodeRed={handleStopNodeRed}
        isRestarting={isRestarting}
        runtime={runtime}
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
