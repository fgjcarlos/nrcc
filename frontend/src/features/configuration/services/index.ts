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
    formData.append('file', file, `${type}-${file.name}`);
    
    return api.post<ApiResponse<{ path: string; filename: string }>>('/files/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 2 * 60_000,
    });
  },
  
  deleteImage: (path: string) => {
    return api.delete<ApiResponse<{ deleted: boolean }>>(`/files/${encodeURIComponent(path)}`);
  },
  
};

export const settingsService = {
  getRaw: () => api.get<ApiResponse<SettingsDocument>>('/settings/raw'),
  saveRaw: (content: string) => api.post<ApiResponse<{ message: string }>>('/settings/raw', { content }),
};
export { aiService } from './aiService';
export type { AIConfig, AIConfigInput, AIProviderStatus } from './aiService';
