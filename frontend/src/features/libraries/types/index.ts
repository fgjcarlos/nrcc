export interface InstalledLibrary {
  name: string;
  alias: string;
  version?: string;
  status: 'active' | 'missing';
  installed: boolean;
}

export interface NpmSearchResult {
  name: string;
  version: string;
  description: string;
  downloads: number;
}

export interface InstallResponse {
  jobId: string;
  message: string;
}
