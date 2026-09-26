import { useDashboardActions, useDashboardData } from '../hooks';
import { DashboardHeader } from './DashboardHeader';
import { DashboardWarnings } from './DashboardWarnings';
import { RestartConfirmationModal } from './RestartConfirmationModal';
import { SecurityPostureTile } from './OverviewTiles/SecurityPostureTile';
import { BackupHealthTile } from './OverviewTiles/BackupHealthTile';
import { NodeRedHealthTile } from './OverviewTiles/NodeRedHealthTile';

/**
 * DashboardView — issue #766 slice D
 *
 * Overview page composition. Slice D replaces the previous generic
 * cards (SystemHealthCard + the metric row + DashboardDetails) with
 * three operational decision tiles:
 *
 *  1. SecurityPostureTile  — adminAuth / httpNodeAuth / httpStaticAuth
 *                            + requireHttps, with a deep link to
 *                            /security.
 *  2. NodeRedHealthTile    — host readiness, runtime status, compact
 *                            resource indicator + restart/open
 *                            actions. Deep link to /configuration.
 *  3. BackupHealthTile     — scheduler health + last backup / last
 *                            automatic / storage. Deep link to
 *                            /backups.
 *
 * Each tile is built on the slice B StatusChip palette; no new visual
 * primitives. The existing DashboardDetails component is removed
 * (its disk card and backup card were either collapsed into the new
 * tiles or absorbed by slice E's Security separation).
 */
export function DashboardView() {
  const { container, system, config, host, runtime, backups, dockerSuccess, dockerLoading, dockerError } = useDashboardData();
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

  // Derive the four security surfaces from the loose-typed config
  // payload that useDashboardData forwards. The cast is intentional —
  // the dashboard layer doesn't yet import NodeRedConfig, but every
  // field below is consumed by ConfigurationView as well.
  const adminAuth = Boolean((config as { adminAuth?: unknown } | undefined)?.adminAuth);
  const httpNodeAuth = Boolean((config as { nodeHttpAuth?: unknown } | undefined)?.nodeHttpAuth);
  const httpStaticAuth = Boolean((config as { staticAuth?: unknown } | undefined)?.staticAuth);
  const requireHttps = Boolean((config as { requireHttps?: unknown } | undefined)?.requireHttps);

  return (
    <div className="space-y-8">
      <DashboardHeader edgeMode={system?.edgeMode} />
      <DashboardWarnings showDockerWarning={showDockerWarning} host={host} />
      <SecurityPostureTile
        adminAuth={adminAuth}
        httpNodeAuth={httpNodeAuth}
        httpStaticAuth={httpStaticAuth}
        requireHttps={requireHttps}
      />
      <NodeRedHealthTile
        host={host}
        runtime={runtime}
        system={system}
        container={container}
        inDocker={inDocker}
        isRestarting={isRestarting}
        onRequestRestart={() => setPendingConfirm(true)}
        onOpenNodeRed={handleOpenNodeRed}
      />
      <BackupHealthTile backups={backups} />
      <RestartConfirmationModal
        isOpen={pendingConfirm}
        onConfirm={handleRestartConfirm}
        onCancel={() => setPendingConfirm(false)}
      />
    </div>
  );
}
