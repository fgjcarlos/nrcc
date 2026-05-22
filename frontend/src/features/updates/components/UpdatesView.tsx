import { useState } from 'react';
import { ConfirmationDialog } from '@/shared/components';
import { Loader2, CheckCircle2, AlertCircle, Database, Zap } from 'lucide-react';
import { useUpdatesData } from '@/features/updates/hooks/useUpdatesData';
import { useUpdatesActions } from '@/features/updates/hooks/useUpdatesActions';
import { formatCheckedAt } from '@/features/updates/lib/updatesFormatters';

export function UpdatesView() {
  const [checkingNow, setCheckingNow] = useState(false);
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
      title: 'Update Node-RED',
      description: `Are you sure you want to update Node-RED from ${status?.currentVersion} to ${status?.latestVersion}?`,
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
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Maintenance</p>
          <h1 className="text-2xl font-bold text-base-content">Node-RED Updates</h1>
        </div>
      </div>

      {/* Status Panel */}
      <div className="surface-card p-6">
        <h2 className="mb-4 text-lg font-semibold text-base-content">Current Status</h2>

        {isLoading && !status ? (
          <div className="space-y-4">
            <div className="flex flex-col items-center justify-center py-8">
              <Loader2 className="w-8 h-8 text-accent animate-spin mb-3" />
              <p className="text-sm text-base-content/60">Checking for updates...</p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {/* Current Version */}
            <div className="flex items-center justify-between">
              <span className="text-base-content/60">Current Version:</span>
              <span className="font-mono font-medium text-base-content">
                {status?.currentVersion || 'unknown'}
              </span>
            </div>

            {/* Latest Version */}
            {hasUpdate && (
              <div className="flex items-center justify-between">
                <span className="text-base-content/60">Latest Version:</span>
                <span className="font-mono font-medium text-base-content">
                  {status?.latestVersion || 'unknown'}
                </span>
              </div>
            )}

            {/* Last Checked */}
            <div className="flex items-center justify-between">
              <span className="text-base-content/60">Last Checked:</span>
              <span className="text-sm text-base-content">{formatCheckedAt(status?.checkedAt)}</span>
            </div>

            {/* Status Badge */}
            <div className="flex items-center justify-between">
              <span className="text-base-content/60">Status:</span>
              {isChecking ? (
                <span className="flex items-center gap-2 text-sm text-accent">
                  <Loader2 className="w-3 h-3 animate-spin" />
                  Checking...
                </span>
              ) : hasError ? (
                <span
                  className="rounded-full bg-error/15 px-2 py-1 text-xs text-error-content"
                  title={status?.error}
                >
                  Error
                </span>
              ) : hasUpdate ? (
                <span className="rounded-full bg-success/15 px-2 py-1 text-xs text-success-content">
                  Update available
                </span>
              ) : (
                <span className="rounded-full bg-base-300/70 px-2 py-1 text-xs text-base-content">
                  Up to date
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
                  Update in Progress
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
                      <p className="text-sm font-medium text-base-content">Create Backup</p>
                      {flowState?.state === 'BackingUp' && (
                        <p className="text-xs text-base-content/60">
                          Backing up Node-RED configuration...
                        </p>
                      )}
                    </div>
                    {flowState?.state === 'BackingUp' && (
                      <span className="text-xs px-2 py-1 rounded-full bg-accent/10 text-accent font-medium">
                        Active
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
                      <p className="text-sm font-medium text-base-content">Update Node-RED</p>
                      {flowState?.state === 'Applying' && (
                        <p className="text-xs text-base-content/60">Installing new version via npm...</p>
                      )}
                    </div>
                    {flowState?.state === 'Applying' && (
                      <span className="text-xs px-2 py-1 rounded-full bg-accent/10 text-accent font-medium">
                        Active
                      </span>
                    )}
                  </div>

                  {/* Status message for backup completion */}
                  {flowState?.state === 'Applying' && flowState?.backupId && (
                    <div className="mt-2 p-2 rounded-lg bg-success/10 border border-success/20 flex items-start gap-2">
                      <CheckCircle2 className="w-4 h-4 text-success flex-shrink-0 mt-0.5" />
                      <p className="text-xs text-success-content">
                        Backup completed (ID: {flowState.backupId.substring(0, 8)}...)
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
                    <p className="font-medium text-success-content">Update completed successfully!</p>
                    <p className="text-sm text-success-content/80 mt-1">
                      Node-RED is now running version {status?.latestVersion}. Some features may
                      require a browser refresh.
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
                    <p className="font-medium text-error-content">Update failed</p>
                    <p className="text-sm text-error-content/80 mt-1">
                      {flowState?.error || 'An error occurred during the update process. Please try again.'}
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
                aria-label="Check for updates"
              >
                {isChecking && <Loader2 className="w-4 h-4 animate-spin" />}
                Check Now
              </button>
              {hasUpdate && !isUpdateActive && (
                <button
                  onClick={handleApplyUpdate}
                  disabled={applyMutation.isPending || isChecking || isUpdateActive}
                  className="action-btn-primary disabled:opacity-50 disabled:cursor-not-allowed"
                  aria-label="Apply update"
                >
                  {applyMutation.isPending ? 'Updating...' : 'Update Now'}
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
        <h2 className="mb-4 text-lg font-semibold text-base-content">Update History</h2>

        {historyLoading ? (
          <div className="animate-pulse">
            <div className="mb-2 h-8 rounded skeleton-dark"></div>
            <div className="mb-2 h-8 rounded skeleton-dark"></div>
            <div className="h-8 rounded skeleton-dark"></div>
          </div>
        ) : history.length === 0 ? (
          <p className="text-sm text-base-content/60">No updates recorded</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b ghost-divider">
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">Date</th>
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">
                    From Version
                  </th>
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">
                    To Version
                  </th>
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">User</th>
                  <th className="px-2 py-3 text-left text-sm font-medium text-base-content">Status</th>
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
                          OK
                        </span>
                      ) : (
                        <span
                          className="rounded-full bg-error/15 px-2 py-1 text-xs text-error-content"
                          title={entry.errorMessage}
                        >
                          Error
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
