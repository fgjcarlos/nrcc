export interface InstalledLibrary {
  name: string;
  version: string;
  description?: string;
  keywords?: string[];
  homepage?: string;
  repository?: string;
}

export interface NpmSearchResult {
  name: string;
  version: string;
  description: string;
  downloads?: number;
}

export interface InstallResponse {
  jobId: string;
  message: string;
}
