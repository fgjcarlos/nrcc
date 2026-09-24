import { useState } from 'react';
import { ConfirmationDialog } from '@/shared/components';
import { Loader2, CheckCircle2, AlertCircle, Database, Zap } from 'lucide-react';
import { useUpdatesData } from '@/features/updates/hooks/useUpdatesData';
import { useUpdatesActions } from '@/features/updates/hooks/useUpdatesActions';
import { formatCheckedAt } from '@/features/updates/lib/updatesFormatters';
import { useT } from '@/i18n';

export function UpdatesView() {
  const [checkingNow, setCheckingNow] = useState(false);
  const { t } = useT();
  const [confirmConfig, setConfirmConfig] = useState<{
    isOpen: boolean;
    title: string;
    description: string;
    confirmText?: string;
    variant: 'danger' | 'warning' | 'default';
    onConfirm: () => void;
  } | null>(null);

  // Data from queries
  const { status, statusLoading, statusRefetch, flowState, history, historyLoading } =
    useUpdatesData();

  // Actions (mutations)
  const { checkMutation, applyMutation } = useUpdatesActions();

  // Local handlers
  const handleCheckNow = async () => {
    setCheckingNow(true);
    try {
      await checkMutation.mutateAsync();
      await statusRefetch();
    } finally {
      setCheckingNow(false);
    }
  };

  const handleApplyUpdate = () => {
    setConfirmConfig({
      isOpen: true,
      title: t('updates:confirmTitle'),
      description: t('updates:confirmDescription', {
        currentVersion: status?.currentVersion ?? t('updates:unknown'),
        latestVersion: status?.latestVersion ?? t('updates:unknown'),
      }),
      variant: 'warning',
      onConfirm: () => {
        setConfirmConfig(null);
        applyMutation.mutate();
      },
    });
  };

  // Derived state
  const isLoading = statusLoading;
  const isChecking = checkingNow || checkMutation.isPending;
  const hasError = status?.error;
  const hasUpdate = status?.updateAvailable && !hasError;

  const isUpdateActive = flowState?.state && ['BackingUp', 'Applying'].includes(flowState.state);
  const isUpdateCompleted = flowState?.state === 'Completed';
  const isUpdateFailed = flowState?.state === 'Failed';

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-between items-center">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">{t('updates:maintenance')}</p>
          <h1 className="text-2xl font-bold text-base-content">{t('updates:title')}</h1>
        </div>
      </div>

      {/* Status Panel */}
      <div className="surface-card p-6">
        <h2 className="mb-4 text-lg font-semibold text-base-content">{t('updates:currentStatus')}</h2>

        {isLoading && !status ? (
          <div className="space-y-4">
            <div className="flex flex-col items-center justify-center py-8">
              <Loader2 className="w-8 h-8 text-accent animate-spin mb-3" />
              <p className="text-sm text-base-content/60">{t('updates:checking')}</p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {/* Current Version */}
            <div className="flex items-center justify-between">
              <span className="text-base-content/60">{t('updates:currentVersion')}</span>
              <span className="font-mono font-medium text-base-content">
                {status?.currentVersion || t('updates:unknown')}
              </span>
            </div>

            {/* Latest Version */}
            {hasUpdate && (
              <div className="flex items-center justify-between">
                <span className="text-base-content/60">{t('updates:latestVersion')}</span>
                <span className="font-mono font-medium text-base-content">
                  {status?.latestVersion || t('updates:unknown')}
                </span>
              </div>
            )}

            {/* Last Checked */}
            <div className="flex items-center justify-between">
              <span className="text-base-content/60">{t('updates:lastChecked')}</span>
              <span className="text-sm text-base-content">{formatCheckedAt(status?.checkedAt)}</span>
            </div>

            {/* Status Badge */}
            <div className="flex items-center justify-between">
              <span className="text-base-content/60">{t('updates:status')}</span>
              {isChecking ? (
                <span className="flex items-center gap-2 text-sm text-accent">
                  <Loader2 className="w-3 h-3 animate-spin" />
                  {t('updates:checking')}
                </span>
              ) : hasError ? (
                <span
                  className="rounded-full bg-error/15 px-2 py-1 text-xs text-error-content"
                  title={status?.error}
                >
                  {t('updates:error')}
                </span>
              ) : hasUpdate ? (
                <span className="rounded-full bg-success/15 px-2 py-1 text-xs text-success-content">
                  {t('updates:updateAvailable')}
                </span>
              ) : (
                <span className="rounded-full bg-base-300/70 px-2 py-1 text-xs text-base-content">
                  {t('updates:upToDate')}
                </span>
              )}
            </div>

            {/* Error message */}
            {hasError && (
              <div className="mt-4 p-3 rounded-lg bg-error/10 border border-error/20">
                <p className="text-sm text-error-content">{status?.error}</p>
              </div>
            )}

            {/* Update Flow Progress (when active) */}
            {isUpdateActive && (
              <div className="mt-6 pt-6 border-t ghost-divider">
                <h3 className="mb-4 text-sm font-semibold text-base-content flex items-center gap-2">
                  <Zap className="w-4 h-4 text-accent" />
                  {t('updates:progress')}
                </h3>

                {/* Step Progress UI */}
                <div className="space-y-3">
                  {/* Step 1: Backup */}
                  <div className="flex items-center gap-3">
                    <div className="flex items-center justify-center w-8 h-8 rounded-full bg-accent/20 text-accent">
                      {flowState?.state === 'BackingUp' ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <Database className="w-4 h-4" />
                      )}
                    </div>
                    <div className="flex-1">
                      <p className="text-sm font-medium text-base-content">{t('updates:createBackup')}</p>
                      {flowState?.state === 'BackingUp' && (
                        <p className="text-xs text-base-content/60">{t('updates:backingUp')}</p>
                      )}
                    </div>
                    {flowState?.state === 'BackingUp' && (
                      <span className="text-xs px-2 py-1 rounded-full bg-accent/10 text-accent font-medium">
                        {t('updates:active')}
                      </span>
                    )}
                  </div>

                  {/* Step 2: Apply */}
                  <div className="flex items-center gap-3">
                    <div className="flex items-center justify-center w-8 h-8 rounded-full bg-accent/20 text-accent">
                      {flowState?.state === 'Applying' ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <Zap className="w-4 h-4" />
                      )}
                    </div>
                    <div className="flex-1">
                      <p className="text-sm font-medium text-base-content">{t('updates:updateNodeRed')}</p>
                      {flowState?.state === 'Applying' && (
                        <p className="text-xs text-base-content/60">{t('updates:installing')}</p>
                      )}
                    </div>
                    {flowState?.state === 'Applying' && (
                      <span className="text-xs px-2 py-1 rounded-full bg-accent/10 text-accent font-medium">
                        {t('updates:active')}
                      </span>
                    )}
                  </div>

                  {/* Status message for backup completion */}
                  {flowState?.state === 'Applying' && flowState?.backupId && (
                    <div className="mt-2 p-2 rounded-lg bg-success/10 border border-success/20 flex items-start gap-2">
                      <CheckCircle2 className="w-4 h-4 text-success flex-shrink-0 mt-0.5" />
                      <p className="text-xs text-success-content">
                        {t('updates:backupCompleted', { id: flowState.backupId.substring(0, 8) })}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Update Completion Feedback */}
            {isUpdateCompleted && (
              <div className="mt-6 pt-6 border-t ghost-divider">
                <div className="p-4 rounded-lg bg-success/10 border border-success/20 flex items-start gap-3">
                  <CheckCircle2 className="w-5 h-5 text-success flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="font-medium text-success-content">{t('updates:completed')}</p>
                    <p className="text-sm text-success-content/80 mt-1">
                      {t('updates:completedDetail', { version: status?.latestVersion ?? t('updates:unknown') })}
                    </p>
                  </div>
                </div>
              </div>
            )}

            {/* Update Error Feedback */}
            {isUpdateFailed && (
              <div className="mt-6 pt-6 border-t ghost-divider">
                <div className="p-4 rounded-lg bg-error/10 border border-error/20 flex items-start gap-3">
                  <AlertCircle className="w-5 h-5 text-error flex-shrink-0 mt-0.5" />
                  <div>
                    <p className="font-medium text-error-content">{t('updates:updateFailed')}</p>
                    <p className="text-sm text-error-content/80 mt-1">
                      {flowState?.error || t('updates:failedDetail')}
                    </p>
                  </div>
                </div>
              </div>
            )}

            {/* Action Buttons */}
            <div className="mt-4 flex gap-2 border-t ghost-divider pt-4">
              <button
                onClick={handleCheckNow}
                disabled={isChecking || isLoading || isUpdateActive}
                className="action-btn-secondary disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                aria-label={t('updates:checkNow')}
              >
                {isChecking && <Loader2 className="w-4 h-4 animate-spin" />}
                {t('updates:checkNow')}
              </button>
              {hasUpdate && !isUpdateActive && (
                <button
                  onClick={handleApplyUpdate}
                  disabled={applyMutation.isPending || isChecking || isUpdateActive}
                  className="action-btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                  aria-label={t('updates:applyUpdate')}
                >
                  {applyMutation.isPending ? t('updates:updating') : t('updates:updateNow')}
                </button>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Confirmation Dialog */}
      {confirmConfig && (
        <ConfirmationDialog
          isOpen={confirmConfig.isOpen}
          title={confirmConfig.title}
          description={confirmConfig.description}
          confirmText={confirmConfig.confirmText}
          variant={confirmConfig.variant}
          isPending={applyMutation.isPending}
          onConfirm={confirmConfig.onConfirm}
          onCancel={() => setConfirmConfig(null)}
        />
      )}

      {/* History Table */}
      <div className="surface-card p-6">
        <h2 className="mb-4 text-lg font-semibold text-base-content">{t('updates:history')}</h2>

        {historyLoading ? (
          <div className="animate-pulse">
            <div className="mb-2 h-8 rounded skeleton-dark"></div>
            <div className="mb-2 h-8 rounded skeleton-dark"></div>
            <div className="h-8 rounded skeleton-dark"></div>
          </div>
        ) : history.length === 0 ? (
          <p className="text-sm text-base-content/60">{t('updates:noHistory')}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b ghost-divider">
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">{t('updates:date')}</th>
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">
                    {t('updates:fromVersion')}
                  </th>
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">
                    {t('updates:toVersion')}
                  </th>
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">{t('updates:user')}</th>
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">{t('updates:status')}</th>
                </tr>
              </thead>
              <tbody>
                {history.map((entry) => (
                  <tr key={entry.id} className="border-b ghost-divider">
                    <td className="px-2 py-3 text-sm text-base-content">
                      {new Date(entry.timestamp).toLocaleString()}
                    </td>
                    <td className="px-2 py-3 font-mono text-sm text-base-content">{entry.fromVersion}</td>
                    <td className="px-2 py-3 font-mono text-sm text-base-content">{entry.toVersion}</td>
                    <td className="px-2 py-3 text-sm text-base-content">{entry.appliedBy}</td>
                    <td className="py-3 px-2">
                      {entry.status === 'success' ? (
                        <span className="rounded-full bg-success/15 px-2 py-1 text-xs text-success-content">
                          {t('common:ok')}
                        </span>
                      ) : (
                        <span
                          className="rounded-full bg-error/15 px-2 py-1 text-xs text-error-content"
                          title={entry.errorMessage}
                        >
                          {t('updates:error')}
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
