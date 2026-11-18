import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { configService, settingsService } from '../services';
import { errorMessage } from '@/shared/lib/errorMessage';
import { queryKeys } from '@/shared/lib/queryKeys';
export function useConfigurationActions() {
  const queryClient = useQueryClient();

  /**
   * Save config mutation.
   *
   * Issue #878: the payload carries the settings.js revision the
   * operator read before composing the edit. The backend rejects a stale
   * revision with 409 SETTINGS_REVISION_CONFLICT instead of overwriting
   * the newer state.
   */
  const saveConfigMutation = useMutation({
    mutationFn: (input: { config: unknown; expectedRevision?: string }) =>
      configService.updateConfig(
        input.config as Record<string, unknown>,
        input.expectedRevision,
      ),
    // No toast: the change circuit's contacts and its polite live region
    // carry the outcome of the gated apply (T9 THESIS).
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.config.root });
      // Issue #878 — the save rewrites settings.js, so the raw-settings
      // document (and with it the revision fingerprint the next save
      // sends) must be refreshed. Without this the operator would send
      // the pre-save revision and the next save would be rejected as
      // stale.
      queryClient.invalidateQueries({ queryKey: queryKeys.config.rawSettings });
    },
  });

  // Save raw settings mutation.
  //
  // Issue #878: the mutation carries the revision of the document the
  // operator opened, so a stale editor session cannot overwrite newer
  // settings.js.
  const saveRawSettingsMutation = useMutation({
    mutationFn: (input: { content: string; expectedRevision?: string }) =>
      settingsService.saveRaw(input.content, input.expectedRevision),
    onSuccess: () => {
      toast.success('settings.js saved');
      queryClient.invalidateQueries({ queryKey: queryKeys.config.rawSettings });
      queryClient.invalidateQueries({ queryKey: queryKeys.bootstrap.status });
      // Issue #880 — a raw save rewrites settings.js, and the backend
      // reconciles the structured configuration with the applied source.
      // The structured configuration query must be refreshed too, or the
      // operator keeps seeing the pre-save configuration.
      queryClient.invalidateQueries({ queryKey: queryKeys.config.root });
    },
    onError: (error) => {
      toast.error(`Failed to save settings.js: ${errorMessage(error)}`);
    },
  });

  /**
   * Handle raw settings save
   *
   * Issue #878: `expectedRevision` is the revision of the document the
   * operator opened. The backend rejects a stale write with 409
   * SETTINGS_REVISION_CONFLICT instead of overwriting newer source.
   */
  const handleSaveRawSettings = (content: string, expectedRevision?: string) => {
    saveRawSettingsMutation.mutate({ content, expectedRevision });
  };

  return {
    // Mutations
    saveConfigMutation,
    saveRawSettingsMutation,

    // Handlers
    handleSaveRawSettings,
  };
}
