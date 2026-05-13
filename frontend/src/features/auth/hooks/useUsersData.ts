import { useQuery } from '@tanstack/react-query';
import { authService, type User } from '@/features/auth/services/authService';

interface UseUsersDataParams {
  enabled?: boolean;
}

export function useUsersData({ enabled = true }: UseUsersDataParams = {}) {
  const usersQuery = useQuery({
    queryKey: ['users'],
    queryFn: authService.getUsers,
    enabled,
  });

  const users = usersQuery.data ?? [];

  return {
    users,
    isLoading: usersQuery.isLoading,
    isError: usersQuery.isError,
    error: usersQuery.error,
    refetch: usersQuery.refetch,
  };
}
