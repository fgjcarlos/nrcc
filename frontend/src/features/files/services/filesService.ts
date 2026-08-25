import api from '@/shared/lib';
import type { ApiResponse } from '@/shared/types';
import type { ManagedFile, UploadedFileResponse } from '../types';

const encodeFileName = (name: string) => encodeURIComponent(name);

export const filesService = {
  list: () => api.get<ApiResponse<ManagedFile[]>>('/files/'),

  upload: (file: File) => {
    const formData = new FormData();
    formData.append('file', file);

    return api.post<ApiResponse<UploadedFileResponse>>('/files/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 2 * 60_000,
    });
  },

  delete: (name: string) => api.delete(`/files/${encodeFileName(name)}`),

  getDownloadUrl: (name: string) => `/api/files/${encodeFileName(name)}/download`,
};
