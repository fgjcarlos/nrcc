import { useMutation, useQueryClient } from '@tanstack/react-query';
import { runtimeService } from '@/features/configuration/services';
import { toast } from 'sonner';

export function useRuntimeActions() {
  const queryClient = useQueryClient();

  const restartMutation = useMutation({
    mutationFn: () => runtimeService.restart(),
    onSuccess: () => {
      toast.success('Runtime restarting...');
      queryClient.invalidateQueries({ queryKey: ['runtime'] });
    },
    onError: () => {
      toast.error('Failed to restart runtime');
    },
  });

  return {
    restartMutation,
  };
}
