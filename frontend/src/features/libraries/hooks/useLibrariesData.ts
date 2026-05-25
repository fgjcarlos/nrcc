import { useQuery } from '@tanstack/react-query';
import { libraryService } from '@/features/libraries/services';

import { queryKeys } from '@/shared/lib/queryKeys';
export function useLibrariesData() {
  const librariesQuery = useQuery({
    queryKey: queryKeys.libraries.root,
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
