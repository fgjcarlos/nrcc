import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { configService, settingsService } from '../services';
import type { NodeRedConfigFormData } from '@/shared/types';
import { formDataToConfigPayload } from '../lib/configTransformers';

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
      toast.error(`Failed to save: ${error}`);
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
      toast.error(`Failed to save settings.js: ${error}`);
    },
  });

  /**
   * Validate authentication fields
   * Returns false if validation fails (toast error already shown)
   */
  const validateAuthFields = (formData: NodeRedConfigFormData): boolean => {
    if (formData.authEnabled) {
      if (!formData.authAdminUser) {
        toast.error('Admin username is required');
        return false;
      }
      if (formData.authAdminUser.length < 3) {
        toast.error('Username must be at least 3 characters');
        return false;
      }
      if (formData.authAdminPassword && formData.authAdminPassword.length < 6) {
        toast.error('Password must be at least 6 characters');
        return false;
      }
    }

    if (formData.authNodeHttpEnabled) {
      if (!formData.authNodeHttpUser || !formData.authNodeHttpPassword) {
        toast.error('Node HTTP username and password are required');
        return false;
      }
      if (formData.authNodeHttpUser.length < 3) {
        toast.error('Node HTTP username must be at least 3 characters');
        return false;
      }
      if (formData.authNodeHttpPassword.length < 6) {
        toast.error('Node HTTP password must be at least 6 characters');
        return false;
      }
    }

    if (formData.authStaticEnabled) {
      if (!formData.authStaticUser || !formData.authStaticPassword) {
        toast.error('Static username and password are required');
        return false;
      }
      if (formData.authStaticUser.length < 3) {
        toast.error('Static username must be at least 3 characters');
        return false;
      }
      if (formData.authStaticPassword.length < 6) {
        toast.error('Static password must be at least 6 characters');
        return false;
      }
    }

    return true;
  };

  /**
   * Handle save with validation
   */
  const handleSave = async (formData: NodeRedConfigFormData) => {
    if (!validateAuthFields(formData)) {
      return;
    }

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
    validateAuthFields,
  };
}
