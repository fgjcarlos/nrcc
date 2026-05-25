import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { authService } from '@/features/auth/services/authService';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useUsersActions() {
  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: ({ username, password, role }: { username: string; password: string; role: 'admin' | 'viewer' }) =>
      authService.createUser(username, password, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.auth.users });
      toast.success('User created successfully');
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string } } } };
      toast.error(err.response?.data?.error?.message || 'Failed to create user');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: authService.deleteUser,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.auth.users });
      toast.success('User deleted successfully');
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string } } } };
      toast.error(err.response?.data?.error?.message || 'Failed to delete user');
    },
  });

  const changePasswordMutation = useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      authService.changePassword(id, password),
    onSuccess: () => {
      toast.success('Password changed successfully');
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string } } } };
      toast.error(err.response?.data?.error?.message || 'Failed to change password');
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: 'admin' | 'viewer' }) =>
      authService.updateUserRole(id, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.auth.users });
      toast.success('User role updated successfully');
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string } } } };
      toast.error(err.response?.data?.error?.message || 'Failed to update user role');
    },
  });

  return {
    createMutation,
    deleteMutation,
    changePasswordMutation,
    updateRoleMutation,
  };
}
