export interface ManagedFile {
  name: string;
  size: number;
  modTime: number;
}

export interface UploadedFileResponse {
  filename: string;
  path: string;
}
