import { api } from 'shared/lib/api';
import type { InstalledLibrary, NpmSearchResult, InstallResponse } from '../types';

export const libraryService = {
  getLibraries: async (): Promise<InstalledLibrary[]> => {
    const response = await api.get<{ data: InstalledLibrary[] }>('/libraries');
    return Array.isArray(response.data.data) ? response.data.data : [];
  },

  searchLibraries: async (query: string): Promise<NpmSearchResult[]> => {
    const response = await api.post<{ data: NpmSearchResult[] }>('/libraries/search', { query });
    return Array.isArray(response.data.data) ? response.data.data : [];
  },

  installLibrary: async (name: string, alias?: string): Promise<InstallResponse> => {
    const response = await api.post<{ data: InstallResponse }>('/libraries/install', {
      name,
      alias: alias || name,
    });
    return response.data.data;
  },

  uninstallLibrary: async (name: string): Promise<void> => {
    await api.delete(`/libraries/${encodeURIComponent(name)}`);
  },

  checkLibrary: async (name: string): Promise<{ name: string; installed: boolean }> => {
    const response = await api.get<{ data: { name: string; installed: boolean } }>(
      `/libraries/${encodeURIComponent(name)}/check`
    );
    return response.data.data;
  },
};
