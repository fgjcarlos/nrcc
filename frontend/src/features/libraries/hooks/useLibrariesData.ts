import { useQuery } from '@tanstack/react-query';
import { libraryService, type InstalledLibrary } from '@/features/libraries/services';

export function useLibrariesData() {
  const librariesQuery = useQuery({
    queryKey: ['libraries'],
    queryFn: libraryService.getLibraries,
  });

  return {
    libraries: librariesQuery.data ?? [],
    isLoading: librariesQuery.isLoading,
    isError: librariesQuery.isError,
    error: librariesQuery.error,
    refetch: librariesQuery.refetch,
  };
}
