import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { configService, settingsService } from '../services';
import type { NodeRedConfigFormData } from '@/shared/types';
import { formDataToConfigPayload } from '../lib/configTransformers';
import { errorMessage } from '@/shared/lib/errorMessage';
import { queryKeys } from '@/shared/lib/queryKeys';
export function useConfigurationActions() {
  const queryClient = useQueryClient();

  // Save config mutation
  const saveConfigMutation = useMutation({
    mutationFn: (config: unknown) =>
      configService.updateConfig(config as Record<string, unknown>),
    onSuccess: () => {
      toast.success('Configuration saved successfully');
      queryClient.invalidateQueries({ queryKey: queryKeys.config.root });
    },
    onError: (error) => {
      toast.error(`Failed to save: ${errorMessage(error)}`);
    },
  });

  // Save raw settings mutation
  const saveRawSettingsMutation = useMutation({
    mutationFn: (content: string) => settingsService.saveRaw(content),
    onSuccess: () => {
      toast.success('settings.js saved');
      queryClient.invalidateQueries({ queryKey: queryKeys.config.rawSettings });
      queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap.status });
    },
    onError: (error) => {
      toast.error(`Failed to save settings.js: ${errorMessage(error)}`);
    },
  });

  /**
   * Handle save with validation
   *
   * The pre-existing `validateAuthFields` aggregate was removed in
   * slice F W2 (issue #766). Per-field validation now flows through
   * `useConfigurationDiff` + `useConfigurationSave` so each
   * `InputField` shows its own `FieldStatusChip` while the operator
   * types instead of getting a single toast on Save.
   */
  const handleSave = async (formData: NodeRedConfigFormData) => {
    const payload = formDataToConfigPayload(formData);
    try {
      await saveConfigMutation.mutateAsync(payload);
    } catch {
      // Error already handled by mutation
    }
  };

  /**
   * Handle raw settings save
   */
  const handleSaveRawSettings = (content: string) => {
    saveRawSettingsMutation.mutate(content);
  };

  return {
    // Mutations
    saveConfigMutation,
    saveRawSettingsMutation,

    // Handlers
    handleSave,
    handleSaveRawSettings,
  };
}
