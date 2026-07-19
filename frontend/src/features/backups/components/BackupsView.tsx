import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { Archive } from 'lucide-react';
import { ConfirmationDialog } from '@/shared/components';
import {
  BackupSummaryCards,
  BackupListSection,
  BackupDetailSection,
  SchedulerConfigSection,
  RetentionPolicySection,
} from '@/features/backups/components';
import { useBackupsData } from '@/features/backups/hooks/useBackupsData';
import { useBackupsActions } from '@/features/backups/hooks/useBackupsActions';
import { backupService, defaultBackupConfig, type BackupConfig, type BackupSummary } from '@/features/backups/services';
import { UI_COPY } from '@/shared/constants/uiCopy';
import {
  getErrorMessage,
  getBackupFileLabel,
  getBackupDisplayName,
} from '@/features/backups/lib/formatters';
import { cn } from '@/shared/lib';

function isConfigDirty(left: BackupConfig, right: BackupConfig | undefined): boolean {
  if (!right) {
    return false;
  }
  return JSON.stringify(left) !== JSON.stringify(right);
}

function getSchedulerTone(status: any): 'healthy' | 'muted' | 'error' {
  if (status.lastError) {
    return 'error';
  }
  if (status.enabled && status.scheduled) {
    return 'healthy';
  }
  return 'muted';
}

function getSchedulerLabel(status: any): string {
  if (status.lastError) {
    return 'Requiere atención';
  }
  if (status.enabled && status.scheduled) {
    return 'Programado';
  }
  return 'Sin programar';
}

type ConfirmConfig = {
  isOpen: boolean;
  title: string;
  description: string;
  confirmText: string;
  variant: 'warning' | 'danger';
  onConfirm: () => void;
} | null;

export function BackupsView() {
  const [configDraft, setConfigDraft] = useState<BackupConfig>(defaultBackupConfig);
  const [selectedBackupId, setSelectedBackupId] = useState<string | null>(null);
  const [confirmConfig, setConfirmConfig] = useState<ConfirmConfig>(null);
  const [backupListPage, setBackupListPage] = useState(1);
  const [backupListSort] = useState<'date' | 'size' | 'status'>('date');
  const [backupListOrder] = useState<'asc' | 'desc'>('desc');

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
  const schedulerTone = getSchedulerTone(effectiveSchedulerStatus);
  const schedulerLabel = getSchedulerLabel(effectiveSchedulerStatus);

  const downloadBackup = async (backup: BackupSummary) => {
    if (!backup.id) {
      toast.error(UI_COPY.backupIdentifierInvalid);
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
      toast.success(UI_COPY.backupDownloadStarted);
    } catch {
      toast.error(UI_COPY.backupDownloadFailed);
    }
  };

  const handleSaveConfig = () => {
    if (configDraft.schedule === 'custom' && !configDraft.customSchedule.trim()) {
      toast.error(UI_COPY.backupCronRequired);
      return;
    }
    actions.saveConfigMutation.mutate(configDraft);
  };

  const handleRestore = (backup: BackupSummary) => {
    const displayName = getBackupDisplayName(backup);
    setConfirmConfig({
      isOpen: true,
      title: 'Restaurar backup',
      description: `¿Querés restaurar ${displayName}? Se generará un backup de seguridad y Node-RED se reiniciará al terminar.`,
      confirmText: backup.id,
      variant: 'warning',
      onConfirm: () => {
        setConfirmConfig(null);
        actions.restoreMutation.mutate(backup.id);
      },
    });
  };

   const handleDelete = (backup: BackupSummary) => {
     const displayName = getBackupDisplayName(backup);
     setConfirmConfig({
       isOpen: true,
       title: UI_COPY.deleteBackup,
       description: UI_COPY.deleteBackupDescription(displayName),
       confirmText: backup.id,
       variant: 'danger',
       onConfirm: () => {
         setConfirmConfig(null);
         actions.deleteMutation.mutate(backup.id);
       },
     });
   };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Backups locales</p>
          <h1 className="text-2xl font-bold text-base-content">Backups</h1>
          <p className="text-sm text-base-content/65">Snapshots locales de flows, settings y archivos clave con restore y retención integrados.</p>
        </div>

        <button
          onClick={() => actions.createMutation.mutate()}
          disabled={actions.createMutation.isPending}
          className="action-btn-primary"
        >
          <Archive className={cn('h-4 w-4', actions.createMutation.isPending && 'animate-pulse')} />
          Crear backup ahora
        </button>
      </div>

      {/* Summary Cards Component */}
      <BackupSummaryCards
        latestBackup={backupsData.backups[0] || null}
        schedulerStatus={effectiveSchedulerStatus}
        schedulerTone={schedulerTone}
        schedulerLabel={schedulerLabel}
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
        onRestore={handleRestore}
        onDelete={handleDelete}
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

      {/* Schedule + Retention Grid */}
      <div className="grid gap-6 md:grid-cols-2">
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
          saveError={actions.saveConfigMutation.error ? getErrorMessage(actions.saveConfigMutation.error, 'Error saving') : undefined}
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

      {confirmConfig && (
        <ConfirmationDialog
          isOpen={confirmConfig.isOpen}
          title={confirmConfig.title}
          description={confirmConfig.description}
          confirmText={confirmConfig.confirmText}
          variant={confirmConfig.variant}
          isPending={actions.restoreMutation.isPending || actions.deleteMutation.isPending}
          onConfirm={confirmConfig.onConfirm}
          onCancel={() => setConfirmConfig(null)}
        />
      )}
    </div>
  );
}
