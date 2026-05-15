import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { useState, useRef, useEffect } from 'react';
import { libraryService, type NpmSearchResult } from '@/features/libraries/services';

export function useLibrariesActions() {
  const queryClient = useQueryClient();
  const [searchResults, setSearchResults] = useState<NpmSearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

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
    // Clear existing timer
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    if (query.length < 2) {
      setSearchResults([]);
      return;
    }

    setSearching(true);

    // Set debounce timer (300ms)
    debounceTimerRef.current = setTimeout(async () => {
      try {
        const results = await libraryService.searchLibraries(query);
        // Limit to first 10 results (backend already does this via size=10)
        setSearchResults(results.slice(0, 10));
      } catch {
        toast.error('Failed to search libraries');
        setSearchResults([]);
      } finally {
        setSearching(false);
      }
    }, 300);
  };

  const handleInstall = (name: string, alias?: string): void => {
    installMutation.mutate({ name, alias });
  };

  const handleUninstall = (name: string): void => {
    uninstallMutation.mutate(name);
  };

  const handleClearSearch = (): void => {
    setSearchResults([]);
  };

  const clearSearchResults = (): void => {
    setSearchResults([]);
  };

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

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
    handleUninstall,
    handleClearSearch,
    clearSearchResults,
  };
}
