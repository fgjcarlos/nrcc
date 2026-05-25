import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { dockerService } from '@/features/docker/services';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useDockerActions() {
  const queryClient = useQueryClient();

  const restartMutation = useMutation({
    mutationFn: () => dockerService.restart(),
    onSuccess: () => {
      toast.success('Container restarting...');
      queryClient.invalidateQueries({ queryKey: queryKeys.docker.root });
    },
    onError: () => {
      toast.error('Failed to restart container');
    },
  });

  const stopMutation = useMutation({
    mutationFn: () => dockerService.stop(),
    onSuccess: () => {
      toast.success('Container stopped');
      queryClient.invalidateQueries({ queryKey: queryKeys.docker.root });
    },
    onError: () => {
      toast.error('Failed to stop container');
    },
  });

  return {
    restartMutation,
    stopMutation,
  };
}
