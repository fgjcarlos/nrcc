import { useQuery } from '@tanstack/react-query';
import { envService } from '@/features/env-vars/services';

export function useEnvVarsData() {
  // Query for all environment variables
  const envVarsQuery = useQuery({
    queryKey: ['envVars'],
    queryFn: envService.getAll,
  });

  // Query for .env file content
  const dotenvQuery = useQuery({
    queryKey: ['envVars-dotenv'],
    queryFn: envService.getDotenv,
  });

  return {
    // Environment variables
    envVars: envVarsQuery.data ?? [],
    isLoading: envVarsQuery.isLoading,
    isError: envVarsQuery.isError,
    error: envVarsQuery.error,

    // .env file
    dotenvContent: dotenvQuery.data?.content ?? '',
    isDotenvLoading: dotenvQuery.isLoading,
    isDotenvError: dotenvQuery.isError,

    // Refetch functions
    refetchEnvVars: envVarsQuery.refetch,
    refetchDotenv: dotenvQuery.refetch,
  };
}
