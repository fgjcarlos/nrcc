import api from '@/shared/lib';
import type { NodeRedConfig, ApiResponse } from '@/shared/types';

export const configService = {
  getConfig: () => api.get<ApiResponse<NodeRedConfig>>('/config'),
  
  updateConfig: (config: Partial<NodeRedConfig>) => 
    api.post<ApiResponse<NodeRedConfig>>('/config', config),
  
  validateConfig: (config: Partial<NodeRedConfig>) => 
    api.post<ApiResponse<{ valid: boolean; errors: string[] }>>('/config/validate', config),
  
  getDefaultConfig: () => 
    api.get<ApiResponse<NodeRedConfig>>('/config/default'),
};

export const runtimeService = {
  getStatus: () => api.get<ApiResponse<{ status: string; uptime: number }>>('/runtime/status'),
  
  restart: () => api.post<ApiResponse<{ message: string; restartInitiated: boolean }>>('/runtime/restart'),
  
  start: () => api.post<ApiResponse<{ message: string }>>('/runtime/start'),
  
  stop: () => api.post<ApiResponse<{ message: string }>>('/runtime/stop'),
  
  getUptime: () => api.get<ApiResponse<{ uptime: number }>>('/runtime/uptime'),
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
  getRaw: () => api.get<ApiResponse<{ content: string }>>('/settings/raw'),
  saveRaw: (content: string) => api.post<ApiResponse<{ message: string }>>('/settings/raw', { content }),
};
