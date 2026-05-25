import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { backupService } from '@/features/backups/services';
import { getErrorMessage } from '@/features/backups/lib/formatters';
import { UI_COPY } from '@/shared/constants/uiCopy';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useBackupsActions() {
  const queryClient = useQueryClient();

  // Save config mutation
  const saveConfigMutation = useMutation({
    mutationFn: backupService.saveConfig,
    onSuccess: async (savedConfig) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.config });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.status });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.observability });
      toast.success('Backup configuration saved');
      return savedConfig;
    },
    onError: (error) => {
      toast.error(getErrorMessage(error, 'Could not save backup configuration'));
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
      toast.success('Backup creado correctamente');
    },
     onError: (error) => {
       toast.error(getErrorMessage(error, UI_COPY.couldNotCreateBackup));
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
      toast.success(result.message || 'Backup restaurado correctamente');
      if (result.preRestoreId) {
        toast.info(`Se creó un backup de seguridad: ${result.preRestoreId}`);
      }
    },
     onError: (error) => {
       toast.error(getErrorMessage(error, UI_COPY.couldNotRestoreBackup));
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
      toast.success('Backup eliminado');
    },
     onError: (error) => {
       toast.error(getErrorMessage(error, UI_COPY.couldNotDeleteBackup));
     },
  });

  // Retention policy mutation
  const retentionMutation = useMutation({
    mutationFn: backupService.patchStorageRetention,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.config });
      await queryClient.invalidateQueries({ queryKey: queryKeys.backups.storage });
      toast.success('Política de retención guardada');
    },
     onError: (error) => {
       toast.error(getErrorMessage(error, UI_COPY.couldNotSaveRetentionPolicy));
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
