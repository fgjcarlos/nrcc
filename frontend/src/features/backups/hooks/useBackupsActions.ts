import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { backupService } from '@/features/backups/services';
import { getErrorMessage } from '@/features/backups/lib/formatters';
import { useT } from '@/i18n';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useBackupsActions() {
  const { t } = useT();

  const queryClient = useQueryClient();

  // Save config mutation
  const saveConfigMutation = useMutation({
    mutationFn: backupService.saveConfig,
    onSuccess: async (savedConfig) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.config });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.status });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.observability });
      toast.success(t('common:backupConfigurationSaved'));
      return savedConfig;
    },
    onError: (error) => {
      toast.error(getErrorMessage(error, t('backups:couldNotSaveBackupConfiguration')));
    },
  });

  // Create backup mutation
  const createMutation = useMutation({
    mutationFn: () => backupService.create('manual'),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.listRoot });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.storage });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.status });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.observability });
      // #483: the detail panel for any backup that may have shifted
      // (pre-restore snapshots, deleted rows) needs to refetch.
      await queryClient.invalidateQueries({ queryKey: ['backup-detail'] });
      toast.success(t('backups:backupCreated'));
    },
     onError: (error) => {
       toast.error(getErrorMessage(error, t('backups:couldNotCreateBackup')));
     },
  });

  // Restore backup mutation
  const restoreMutation = useMutation({
    mutationFn: backupService.restore,
    onSuccess: async (result) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.listRoot });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.storage });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.status });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.observability });
      // #483: restore may create or modify related rows (pre-restore
      // snapshot); refetch any open detail panel.
      await queryClient.invalidateQueries({ queryKey: ['backup-detail'] });
      toast.success(result.message || t('backups:backupRestored'));
      if (result.preRestoreId) {
        toast.info(t('backups:preRestoreBackupNotice', { id: result.preRestoreId }));
      }
    },
     onError: (error) => {
       toast.error(getErrorMessage(error, t('backups:couldNotRestoreBackup')));
     },
  });

  // Delete backup mutation
  const deleteMutation = useMutation({
    mutationFn: backupService.delete,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.listRoot });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.storage });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.status });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.observability });
      // #483: the deleted row's detail panel needs to refetch so it
      // surfaces the missing entry instead of a stale manifest.
      await queryClient.invalidateQueries({ queryKey: ['backup-detail'] });
      toast.success(t('backups:backupDeleted'));
    },
     onError: (error) => {
       toast.error(getErrorMessage(error, t('backups:couldNotDeleteBackup')));
     },
  });

  // Retention policy mutation
  const retentionMutation = useMutation({
    mutationFn: backupService.patchStorageRetention,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.config });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.storage });
      toast.success(t('backups:retentionPolicySaved'));
    },
     onError: (error) => {
       toast.error(getErrorMessage(error, t('backups:couldNotSaveRetentionPolicy')));
     },
  });

  return {
    saveConfigMutation,
    createMutation,
    restoreMutation,
    deleteMutation,
    retentionMutation,
  };
}
