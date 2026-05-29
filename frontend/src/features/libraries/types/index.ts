export interface InstalledLibrary {
  name: string;
  version: string;
  description?: string;
  keywords?: string[];
  category?: string;
  author?: string;
  license?: string;
  homepage?: string;
  repository?: string;
  npm?: string;
  downloads?: number;
  date?: string;
}

export type NpmSearchResult = InstalledLibrary;

export interface InstallResponse {
  jobId: string;
  message: string;
}
