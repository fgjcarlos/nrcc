import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { envService } from '@/features/env-vars/services';

export function useEnvVarsActions() {
  const queryClient = useQueryClient();

  // Create environment variable mutation
  const createMutation = useMutation({
    mutationFn: envService.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['envVars'] });
      toast.success('Variable created successfully');
    },
    onError: (error: unknown) => {
      let message = 'Failed to create variable';
      if (error && typeof error === 'object' && 'response' in error) {
        const err = error as { response?: { data?: { error?: { message?: string } } } };
        if (err.response?.data?.error?.message) {
          message = err.response.data.error.message;
        }
      }
      toast.error(message);
    },
  });

  // Delete environment variable mutation
  const deleteMutation = useMutation({
    mutationFn: envService.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['envVars'] });
      toast.success('Variable deleted successfully');
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string } } } };
      toast.error(err.response?.data?.error?.message || 'Failed to delete variable');
    },
  });

  // Save .env file mutation
  const saveDotenvMutation = useMutation({
    mutationFn: envService.saveDotenv,
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['envVars-dotenv'] });
      queryClient.invalidateQueries({ queryKey: ['envVars'] });
      toast.success(result.message || '.env file saved successfully');
      if (result.restarted) {
        toast.info('Node-RED restarted');
      }
    },
    onError: (error: unknown) => {
      let message = 'Failed to save .env file';
      if (error && typeof error === 'object' && 'response' in error) {
        const err = error as { response?: { data?: { error?: { message?: string } } } };
        if (err.response?.data?.error?.message) {
          message = err.response.data.error.message;
        }
      }
      toast.error(message);
    },
  });

  return {
    createMutation,
    deleteMutation,
    saveDotenvMutation,
  };
}
