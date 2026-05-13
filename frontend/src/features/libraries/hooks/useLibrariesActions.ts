import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { useState } from 'react';
import { libraryService, type NpmSearchResult } from '@/features/libraries/services';

export function useLibrariesActions() {
  const queryClient = useQueryClient();
  const [searchResults, setSearchResults] = useState<NpmSearchResult[]>([]);
  const [searching, setSearching] = useState(false);

  const installMutation = useMutation({
    mutationFn: ({ name, alias }: { name: string; alias?: string }) =>
      libraryService.installLibrary(name, alias),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['libraries'] });
      toast.success('Library installed successfully');
      setSearchResults([]);
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string } } } };
      toast.error(err.response?.data?.error?.message || 'Failed to install library');
    },
  });

  const uninstallMutation = useMutation({
    mutationFn: libraryService.uninstallLibrary,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['libraries'] });
      toast.success('Library uninstalled successfully');
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string } } } };
      toast.error(err.response?.data?.error?.message || 'Failed to uninstall library');
    },
  });

  const handleSearch = async (query: string): Promise<void> => {
    if (query.length < 2) {
      setSearchResults([]);
      return;
    }

    setSearching(true);
    try {
      const results = await libraryService.searchLibraries(query);
      setSearchResults(results);
    } catch {
      toast.error('Failed to search libraries');
    } finally {
      setSearching(false);
    }
  };

  const handleInstall = (name: string, alias?: string): void => {
    installMutation.mutate({ name, alias });
  };

  const handleClearSearch = (): void => {
    setSearchResults([]);
  };

  const clearSearchResults = (): void => {
    setSearchResults([]);
  };

  return {
    // Mutations
    installMutation,
    uninstallMutation,
    
    // Search state
    searchResults,
    searching,
    
    // Handlers
    handleSearch,
    handleInstall,
    handleClearSearch,
    clearSearchResults,
  };
}
