import { useState, type ReactNode } from 'react';
import { History, Loader2 } from 'lucide-react';
import { ConfirmationDialog } from '@/shared/components';
import { EmptyState, Skeleton } from '@/shared/components/feedback';
import { Button, StatusChip, Table, type StatusChipProps } from '@/shared/components/ui';
import { useUpdatesData } from '@/features/updates/hooks/useUpdatesData';
import { useUpdatesActions } from '@/features/updates/hooks/useUpdatesActions';
import { formatCheckedAt } from '@/features/updates/lib/updatesFormatters';
import { useT } from '@/i18n';

const knownVersionPattern = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/;
const isKnownVersion = (version: string | undefined) => !!version && knownVersionPattern.test(version);

type CheckState = 'checking' | 'failed' | 'available' | 'current' | 'unknown';
const CHECK_CHIP: Record<CheckState, StatusChipProps['variant']> = {
  checking: 'neutral',
  failed: 'danger',
  available: 'info',
  current: 'success',
  unknown: 'neutral',
};

const designator = 'font-mono text-xs font-semibold tabular-nums text-ds-text-primary';

export function UpdatesView() {
  const [checkingNow, setCheckingNow] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const { t, i18n } = useT();

  const { status, statusLoading, statusRefetch, flowState, history, historyLoading } = useUpdatesData();
  const { checkMutation, applyMutation } = useUpdatesActions();

  const handleCheckNow = async () => {
    setCheckingNow(true);
    try {
      await checkMutation.mutateAsync();
      await statusRefetch();
    } finally {
      setCheckingNow(false);
    }
  };

  const isChecking = checkingNow || checkMutation.isPending;
  const hasError = !!status?.error;
  const hasUpdate = !!status?.updateAvailable && !hasError;
  const currentVersion = status?.currentVersion;
  const latestVersion = status?.latestVersion;
  const hasKnownVersions = isKnownVersion(currentVersion) && isKnownVersion(latestVersion);
  const canApplyUpdate = status?.strategy === 'npm-global' && status.canInplaceApply === true && hasKnownVersions && hasUpdate;

  // "Up to date" only when both versions are real; anything else is unconfirmed.
  const checkState: CheckState =
    isChecking || (statusLoading && !status)
      ? 'checking'
      : hasError
        ? 'failed'
        : hasUpdate
          ? 'available'
          : hasKnownVersions
            ? 'current'
            : 'unknown';

  const updateGuidance = status?.strategy === 'image-local'
    ? t('updates:imageGuidance')
    : status?.strategy === 'external'
      ? t('updates:externalGuidance')
      : status?.strategy === 'npm-global' && status.canInplaceApply === true
        ? t('updates:npmGuidance')
        : t('updates:unknownGuidance');

  const flow = flowState?.state;
  const isUpdateActive = flow === 'BackingUp' || flow === 'Applying';
  const targetVersion = latestVersion || t('updates:unknown');
  const failureCopy = flowState?.error === 'image_unsupported' || flowState?.error === 'external_unsupported'
    ? t(`updates:${flowState.error}`)
    : flowState?.error === 'backup_failed' || flowState?.error === 'internal_error'
      ? t(`updates:failed.${flowState.error}`)
      : t('updates:failed.update_failed');

  const versionValue = (version: string | undefined) =>
    isKnownVersion(version)
      ? <span className={designator}>{version}</span>
      : <span className="text-sm text-ds-text-secondary">{t('updates:unknown')}</span>;

  return (
    <div className="space-y-4">
      <header className="flex flex-col gap-3 border-b border-ds-border-subtle pb-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0 space-y-1">
          <h1 className="text-xl font-bold text-ds-text-primary">{t('updates:title')}</h1>
          <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('updates:intro')}</p>
        </div>
        <Button
          type="button"
          variant="secondary"
          onClick={handleCheckNow}
          disabled={isChecking || statusLoading || isUpdateActive}
          className="min-h-11 shrink-0 self-start whitespace-nowrap md:min-h-9 sm:self-auto"
        >
          {isChecking && <Loader2 className="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />}
          {t('updates:checkNow')}
        </Button>
      </header>

      {/* One sheet: sections separated by hairline rules, no nested cards. */}
      <div className="divide-y divide-ds-border-subtle">
        <section aria-labelledby="updates-runtime-title" className="space-y-3 py-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 id="updates-runtime-title" className="text-sm font-semibold text-ds-text-primary">
              {t('updates:runtimeTitle')}
            </h2>
            <StatusChip
              size="sm"
              variant={CHECK_CHIP[checkState]}
              icon={checkState === 'checking' ? <Loader2 className="h-3 w-3 animate-spin motion-reduce:animate-none" /> : undefined}
            >
              {t(`updates:state.${checkState}`)}
            </StatusChip>
          </div>

          <dl className="grid gap-x-8 gap-y-3 sm:grid-cols-3">
            <Fact label={t('updates:installed')}>{versionValue(currentVersion)}</Fact>
            <Fact label={t('updates:latestOnNpm')}>{versionValue(latestVersion)}</Fact>
            <Fact label={t('updates:lastChecked')}>
              <span className="text-sm text-ds-text-primary">{formatCheckedAt(status?.checkedAt, t, i18n.language)}</span>
            </Fact>
          </dl>

          {hasError && !isChecking && (
            <div className="space-y-1 text-sm">
              <p className="text-ds-on-tint-error">{t('updates:checkFailedDetail')}</p>
              <p className="text-xs text-ds-text-secondary">{status?.error}</p>
            </div>
          )}

          <p className="max-w-[75ch] text-sm text-ds-text-secondary" role="note">{updateGuidance}</p>

          {canApplyUpdate && !isUpdateActive && (
            <Button
              type="button"
              variant="solid"
              onClick={() => setConfirmOpen(true)}
              disabled={applyMutation.isPending || isChecking}
              className="min-h-11 md:min-h-9"
            >
              {applyMutation.isPending ? t('updates:updating') : t('updates:updateTo', { version: latestVersion })}
            </Button>
          )}
        </section>

        {isUpdateActive && (
          <section aria-labelledby="updates-progress-title" aria-live="polite" className="space-y-3 py-4">
            <h2 id="updates-progress-title" className="text-sm font-semibold text-ds-text-primary">
              {t('updates:progress')}
            </h2>
            <ol className="space-y-2 text-sm">
              <Step
                label={t('updates:steps.backup')}
                state={flow === 'BackingUp' ? 'running' : 'done'}
                detail={flowState?.backupId && (
                  <>
                    {t('updates:steps.backupId')} <span className={designator}>{flowState.backupId.substring(0, 8)}</span>
                  </>
                )}
              />
              <Step
                label={t('updates:steps.install', { version: flowState?.availableVersion || targetVersion })}
                state={flow === 'Applying' ? 'running' : 'waiting'}
              />
            </ol>
          </section>
        )}

        {flow === 'Completed' && (
          <section role="status" className="space-y-1 py-4">
            <h2 className="text-sm font-semibold text-ds-on-tint-success">
              {t('updates:completedTitle', { version: targetVersion })}
            </h2>
            <p className="max-w-[75ch] text-sm text-ds-text-secondary">
              {t('updates:completedDetail', { version: targetVersion })}
            </p>
          </section>
        )}

        {flow === 'Failed' && (
          <section role="alert" className="space-y-1 py-4">
            <h2 className="text-sm font-semibold text-ds-on-tint-error">{t('updates:updateFailed')}</h2>
            <p className="max-w-[75ch] text-sm text-ds-text-secondary">{failureCopy}</p>
          </section>
        )}

        <section aria-labelledby="updates-history-title" className="space-y-3 py-4">
          <div className="space-y-1">
            <h2 id="updates-history-title" className="text-sm font-semibold text-ds-text-primary">
              {t('updates:historyTitle')}
            </h2>
            <p className="text-xs text-ds-text-secondary">{t('updates:historyHint')}</p>
          </div>

          {historyLoading ? (
            <Skeleton rows={3} className="h-8" />
          ) : history.length === 0 ? (
            <EmptyState
              icon={<History className="h-6 w-6" />}
              title={t('updates:historyEmpty')}
              description={t('updates:historyEmptyHint')}
            />
          ) : (
            <Table stacked caption={t('updates:historyTitle')}>
              <thead>
                <tr>
                  <th>{t('updates:date')}</th>
                  <th>{t('updates:fromVersion')}</th>
                  <th>{t('updates:backup')}</th>
                </tr>
              </thead>
              <tbody>
                {history.map((entry) => (
                  <tr key={entry.id}>
                    <td data-label={t('updates:date')} className="tabular-nums text-ds-text-secondary">
                      {new Date(entry.timestamp).toLocaleString(i18n.language)}
                    </td>
                    <td data-label={t('updates:fromVersion')}>
                      <span className={designator}>{entry.fromVersion}</span>
                    </td>
                    <td data-label={t('updates:backup')}>
                      <span className={designator} title={entry.id}>{entry.id.substring(0, 8)}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          )}
        </section>
      </div>

      <ConfirmationDialog
        isOpen={confirmOpen}
        title={t('updates:confirmTitle', { version: targetVersion })}
        description={t('updates:confirmDescription', {
          currentVersion: currentVersion ?? t('updates:unknown'),
          latestVersion: targetVersion,
        })}
        variant="warning"
        isPending={applyMutation.isPending}
        onConfirm={() => {
          setConfirmOpen(false);
          applyMutation.mutate();
        }}
        onCancel={() => setConfirmOpen(false)}
      />
    </div>
  );
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1">
      <dt className="text-xs text-ds-text-secondary">{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

function Step({ label, state, detail }: { label: string; state: 'waiting' | 'running' | 'done'; detail?: ReactNode }) {
  const { t } = useT();
  return (
    <li className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <span className="text-ds-text-primary">{label}</span>
      <span className="flex items-baseline gap-3 text-xs">
        {detail && <span className="text-ds-text-secondary">{detail}</span>}
        <span
          className={
            state === 'done'
              ? 'font-semibold text-ds-on-tint-success'
              : state === 'running'
                ? 'font-semibold text-ds-on-tint-info'
                : 'text-ds-text-muted'
          }
        >
          {t(`updates:steps.${state}`)}
        </span>
      </span>
    </li>
  );
}
