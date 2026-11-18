import { useT } from '@/i18n';
import { useDashboardActions, useDashboardData } from '../hooks';
import { backupState, formatAge, postureFromConfig, runtimeKey, securityVerdict, summaryAction } from '../lib';
import { DashboardHeader } from './DashboardHeader';
import { DashboardWarnings } from './DashboardWarnings';
import { RestartConfirmationModal } from './RestartConfirmationModal';
import { SecurityPostureTile } from './OverviewTiles/SecurityPostureTile';
import { BackupHealthTile } from './OverviewTiles/BackupHealthTile';
import { NodeRedHealthTile } from './OverviewTiles/NodeRedHealthTile';

/**
 * Overview (T8): one summary sentence and at most one next action, then the
 * sections (Node-RED, security, backups) in the sentence's order, separated
 * by hairline rules on the layout's single sheet (no nested cards).
 */
export function DashboardView() {
  const { t, i18n } = useT();
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

  const posture = postureFromConfig(config);
  const verdict = posture && securityVerdict(posture);
  const backup = backupState(backups);
  const runtimeStatus = runtimeKey(runtime?.status ?? (host?.nodeRed.running ? 'running' : undefined));
  const summary = t('dashboard:summary.sentence', {
    runtime: t(`dashboard:summary.runtime.${runtimeStatus}`),
    security: t(`dashboard:summary.security.${verdict ?? 'unknown'}`),
    backup: backup.kind === 'age'
      ? t('dashboard:summary.backup.age', { age: formatAge(backup.createdAt, i18n.language) })
      : t(`dashboard:summary.backup.${backup.kind}`),
  });

  return (
    <div className="space-y-4">
      <DashboardHeader summary={summary} action={summaryAction(verdict, backup)} />
      <DashboardWarnings showDockerWarning={showDockerWarning} host={host} />
      <div className="divide-y divide-ds-border-subtle">
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
        <SecurityPostureTile {...posture} />
        <BackupHealthTile backups={backups} />
      </div>
      <RestartConfirmationModal
        isOpen={pendingConfirm}
        onConfirm={handleRestartConfirm}
        onCancel={() => setPendingConfirm(false)}
      />
    </div>
  );
}
