import api from '@/shared/lib';
import type { NodeRedConfig, ApiResponse, SettingsDocument } from '@/shared/types';

export const configService = {
  getConfig: () => api.get<ApiResponse<NodeRedConfig>>('/config'),
  
  updateConfig: (config: Record<string, unknown>) =>
    api.post<ApiResponse<NodeRedConfig>>('/config', config),
  
  validateConfig: (config: Partial<NodeRedConfig>) => 
    api.post<ApiResponse<{ valid: boolean; errors: string[] }>>('/config/validate', config),
  
  getDefaultConfig: () => 
    api.get<ApiResponse<NodeRedConfig>>('/config/default'),
};

export const fileService = {
  uploadImage: (type: 'favicon' | 'header' | 'login', file: File) => {
    const formData = new FormData();
    formData.append('image', file);
    
    return api.post<ApiResponse<{ path: string; url: string; filename: string }>>(`/files/upload/${type}`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
  
  deleteImage: (path: string) => {
    return api.delete<ApiResponse<{ deleted: boolean }>>(`/files/${encodeURIComponent(path)}`);
  },
  
  listImages: (type: 'favicon' | 'header' | 'login') => {
    return api.get<ApiResponse<string[]>>(`/files/list/${type}`);
  },
};

export const settingsService = {
  getRaw: () => api.get<ApiResponse<SettingsDocument>>('/settings/raw'),
  saveRaw: (content: string) => api.post<ApiResponse<{ message: string }>>('/settings/raw', { content }),
};
