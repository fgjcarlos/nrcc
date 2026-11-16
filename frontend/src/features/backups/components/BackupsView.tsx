import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { Archive, RefreshCw, AlertTriangle } from 'lucide-react';
import { Button } from '@/shared/components/ui';
import {
  BackupSummaryCards,
  BackupListSection,
  BackupDetailSection,
  SchedulerConfigSection,
  RetentionPolicySection,
  BackupActionDialog,
  type BackupAction,
  type SchedulerState,
} from '@/features/backups/components';
import { useBackupsData } from '@/features/backups/hooks/useBackupsData';
import { useBackupsActions } from '@/features/backups/hooks/useBackupsActions';
import {
  backupService,
  defaultBackupConfig,
  type BackupConfig,
  type BackupSchedulerStatus,
  type BackupSummary,
} from '@/features/backups/services';
import { useT } from '@/i18n';
import { getErrorMessage, getBackupFileLabel } from '@/features/backups/lib/formatters';
import { cn } from '@/shared/lib';

function isConfigDirty(left: BackupConfig, right: BackupConfig | undefined): boolean {
  if (!right) {
    return false;
  }
  return JSON.stringify(left) !== JSON.stringify(right);
}

/** A failed run is the only red state; unscheduled is a held gate (amber). */
function getSchedulerState(status: BackupSchedulerStatus): SchedulerState {
  if (status.lastError) {
    return 'failed';
  }
  if (status.enabled && status.scheduled) {
    return 'scheduled';
  }
  return 'unscheduled';
}

type PendingAction = { action: BackupAction; backup: BackupSummary } | null;

export function BackupsView() {
  const [configDraft, setConfigDraft] = useState<BackupConfig>(defaultBackupConfig);
  const [selectedBackupId, setSelectedBackupId] = useState<string | null>(null);
  const [pendingAction, setPendingAction] = useState<PendingAction>(null);
  const [backupListPage, setBackupListPage] = useState(1);
  const [backupListSort] = useState<'date' | 'size' | 'status'>('date');
  const [backupListOrder] = useState<'asc' | 'desc'>('desc');

  const { t } = useT();


  const backupsData = useBackupsData({
    page: backupListPage,
    limit: 10,
    sort: backupListSort,
    order: backupListOrder,
    selectedBackupId,
  });

  const actions = useBackupsActions();

  // Sync config from query to draft — but only when the operator has no
  // pending edits. `saveConfig` / `retention` invalidate the config query,
  // and a refetch mid-edit would silently overwrite the draft. We compute
  // `configDirty` here (rather than at the JSX site below) so this guard
  // sees the current value, not a stale reference.
  const configDirty = isConfigDirty(configDraft, backupsData.config);

  useEffect(() => {
    if (backupsData.config && !configDirty) {
      setConfigDraft(backupsData.config);
    }
  }, [backupsData.config, configDirty]);

  // Clear selected backup if not in list
  useEffect(() => {
    if (selectedBackupId && !backupsData.backups.some((backup) => backup.id === selectedBackupId)) {
      setSelectedBackupId(null);
    }
  }, [backupsData.backups, selectedBackupId]);

  const selectedBackup = useMemo(
    () => backupsData.backups.find((backup) => backup.id === selectedBackupId) || null,
    [backupsData.backups, selectedBackupId]
  );

  const storageSummary = useMemo(
    () => ({
      totalBackups: backupsData.backups.length,
      totalSize: backupsData.backups.reduce((sum, backup) => sum + (Number.isFinite(backup.totalSize) ? backup.totalSize : 0), 0),
      manualCount: backupsData.backups.filter((backup) => backup.type === 'manual').length,
      autoCount: backupsData.backups.filter((backup) => backup.type === 'auto').length,
      preRestoreCount: backupsData.backups.filter((backup) => backup.type === 'pre-restore').length,
    }),
    [backupsData.backups]
  );

  const effectiveStorage = backupsData.observability?.storage ?? backupsData.storage ?? storageSummary;
  const pendingActionId = actions.restoreMutation.variables ?? actions.deleteMutation.variables ?? null;
  const selectedDetailLoading = Boolean(selectedBackupId) && (backupsData.detailLoading || backupsData.detailFetching || backupsData.detail === undefined);
  const effectiveSchedulerStatus = backupsData.status ?? {
    enabled: configDraft.enabled,
    scheduled: configDraft.enabled && configDraft.schedule !== 'disabled',
    schedule: configDraft.schedule,
    customSchedule: configDraft.customSchedule,
    activeSpec: configDraft.schedule === 'custom' ? configDraft.customSchedule : '',
    nextRunAt: '',
    lastRunAt: '',
    lastSuccessAt: '',
    lastBackupId: undefined,
    lastError: undefined,
  };
  const schedulerState = getSchedulerState(effectiveSchedulerStatus);

  // #483: surface a sync error banner when the live status / observability
  // queries failed. The cards below still render with cached or fallback
  // data, but the operator now sees that the view is stale and can retry.
  const dataSyncError: 'status' | 'observability' | null = backupsData.statusError
    ? 'status'
    : backupsData.observabilityError
      ? 'observability'
      : null;

  const downloadBackup = async (backup: BackupSummary) => {
    if (!backup.id) {
      toast.error(t('backups:backupIdentifierInvalid'));
      return;
    }

    try {
      const blob = await backupService.download(backup.id);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = getBackupFileLabel(backup);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      toast.success(t('backups:backupDownloadStarted'));
    } catch {
      toast.error(t('backups:backupDownloadFailed'));
    }
  };

  const handleSaveConfig = () => {
    if (configDraft.schedule === 'custom' && !configDraft.customSchedule.trim()) {
      toast.error(t('backups:backupCronRequired'));
      return;
    }
    actions.saveConfigMutation.mutate(configDraft);
  };

  const confirmPendingAction = () => {
    if (!pendingAction) return;
    const { action, backup } = pendingAction;
    setPendingAction(null);
    if (action === 'restore') {
      actions.restoreMutation.mutate(backup.id);
    } else {
      actions.deleteMutation.mutate(backup.id);
    }
  };

  return (
    <div className="space-y-4">
      <header className="flex flex-col gap-3 border-b border-ds-border-subtle pb-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0 space-y-1">
          <h1 className="text-xl font-bold text-ds-text-primary">{t('backups:backupsTitle')}</h1>
          <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('backups:localBackupsSubtitle')}</p>
        </div>

        <Button
          type="button"
          onClick={() => actions.createMutation.mutate()}
          disabled={actions.createMutation.isPending}
          className="min-h-11 shrink-0 self-start whitespace-nowrap sm:min-h-9 sm:self-auto"
        >
          <Archive
            className={cn('h-4 w-4', actions.createMutation.isPending && 'animate-pulse motion-reduce:animate-none')}
            aria-hidden="true"
          />
          {t('backups:createBackupNow')}
        </Button>
      </header>

      {dataSyncError && (
        <div
          role="alert"
          className="flex flex-wrap items-start gap-3 rounded-box border border-ds-warning/35 bg-ds-warning/10 px-3 py-2"
        >
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-ds-on-tint-warning" aria-hidden="true" />
          <div className="min-w-0 flex-1 text-sm">
            <p className="font-semibold text-ds-text-primary">
              {t(dataSyncError === 'status' ? 'backups:syncError.status' : 'backups:syncError.observability')}
            </p>
            <p className="mt-1 text-ds-text-secondary">{t('backups:syncError.body')}</p>
          </div>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            onClick={() => {
              // Resync all four backup-related queries without a full page
              // reload — see issue #590 and auditoria/frontend.md HIGH-5.
              void backupsData.refetchConfig();
              void backupsData.refetchStatus();
              void backupsData.refetchObservability();
              void backupsData.refetchStorage();
            }}
          >
            <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
            {t('backups:syncError.retry')}
          </Button>
        </div>
      )}

      {/* One sheet: sections separated by hairline rules, no nested cards. */}
      <div className="divide-y divide-ds-border-subtle">
        <BackupSummaryCards
          latestBackup={backupsData.backups[0] || null}
          schedulerStatus={effectiveSchedulerStatus}
          schedulerState={schedulerState}
          storage={{
            totalBackups: effectiveStorage.totalBackups,
            totalSize: effectiveStorage.totalSize,
          }}
        />

        {/* Backup List Section Component */}
        <BackupListSection
          backups={backupsData.backups}
          isLoading={backupsData.backupsLoading}
          isError={backupsData.backupsError}
          selectedBackupId={selectedBackupId}
          total={backupsData.backupList?.total ?? 0}
          page={backupListPage}
          isActionPending={actions.restoreMutation.isPending || actions.deleteMutation.isPending}
          pendingActionId={pendingActionId}
          onSelect={setSelectedBackupId}
          onDownload={downloadBackup}
          onRestore={(backup) => setPendingAction({ action: 'restore', backup })}
          onDelete={(backup) => setPendingAction({ action: 'delete', backup })}
          onRetry={() => void backupsData.refetchBackups()}
          onPageChange={setBackupListPage}
          isCreating={actions.createMutation.isPending}
          onCreateBackup={() => actions.createMutation.mutate()}
        />

        {/* Backup Detail Section Component */}
        {selectedBackup && (
          <BackupDetailSection
            backup={selectedBackup}
            detail={backupsData.detail}
            isLoading={selectedDetailLoading}
            onClose={() => setSelectedBackupId(null)}
          />
        )}

        {/* Schedule + Retention */}
        <div className="grid gap-x-8 md:grid-cols-2 max-md:divide-y max-md:divide-ds-border-subtle">
          {/* LEFT COLUMN: Schedule */}
          <SchedulerConfigSection
            configDraft={configDraft}
            saveState={
              actions.saveConfigMutation.isPending
                ? 'saving'
                : actions.saveConfigMutation.isError
                  ? 'error'
                  : configDirty
                    ? 'idle'
                    : 'saved'
            }
            saveError={actions.saveConfigMutation.error ? getErrorMessage(actions.saveConfigMutation.error, t('backups:couldNotSaveBackupConfiguration')) : undefined}
            onChange={(cron) => {
              setConfigDraft((current) => ({
                ...current,
                customSchedule: cron,
              }));
            }}
            onPresetChange={(preset) => {
              setConfigDraft((current) => {
                const newSchedule = preset || 'disabled';
                return {
                  ...current,
                  schedule: newSchedule as BackupConfig['schedule'],
                  enabled: newSchedule !== 'disabled',
                  customSchedule: newSchedule === 'custom' ? current.customSchedule : '',
                };
              });
            }}
            onSave={() => handleSaveConfig()}
          />

          {/* RIGHT COLUMN: Retention */}
          <RetentionPolicySection
            retentionManual={configDraft.retentionManual}
            retentionAuto={configDraft.retentionAuto}
            retentionPreRestore={configDraft.retentionPreRestore}
            isSaving={actions.retentionMutation.isPending}
            onSave={(manual, auto, preRestore) => {
              const updatedConfig = {
                ...configDraft,
                retentionManual: manual,
                retentionAuto: auto,
                retentionPreRestore: preRestore,
              };
              setConfigDraft(updatedConfig);
              actions.retentionMutation.mutate(manual);
            }}
          />
        </div>
      </div>

      {pendingAction && (
        <BackupActionDialog
          key={`${pendingAction.action}-${pendingAction.backup.id}`}
          action={pendingAction.action}
          backup={pendingAction.backup}
          isPending={actions.restoreMutation.isPending || actions.deleteMutation.isPending}
          onConfirm={confirmPendingAction}
          onCancel={() => setPendingAction(null)}
        />
      )}
    </div>
  );
}
