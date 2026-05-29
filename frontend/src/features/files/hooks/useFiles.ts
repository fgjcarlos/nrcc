import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { queryKeys } from '@/shared/lib/queryKeys';
import { filesService } from '../services';

export function useFiles() {
  const queryClient = useQueryClient();

  const filesQuery = useQuery({
    queryKey: queryKeys.files.root,
    queryFn: () => filesService.list(),
  });

  const uploadMutation = useMutation({
    mutationFn: (file: File) => filesService.upload(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.files.root });
      toast.success('File uploaded successfully');
    },
    onError: () => toast.error('Failed to upload file'),
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => filesService.delete(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.files.root });
      toast.success('File deleted successfully');
    },
    onError: () => toast.error('Failed to delete file'),
  });

  return {
    files: filesQuery.data?.data?.data ?? [],
    isLoading: filesQuery.isLoading,
    isError: filesQuery.isError,
    error: filesQuery.error,
    refetch: filesQuery.refetch,
    uploadMutation,
    deleteMutation,
  };
}
