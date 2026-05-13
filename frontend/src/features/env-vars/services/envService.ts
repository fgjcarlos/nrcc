import { api } from '@/shared/lib';

export interface EnvVar {
  key: string;
  value: string;
  type: 'string' | 'number' | 'boolean' | 'secret';
  description?: string;
  encrypted?: boolean;
}

export const envService = {
  getAll: async (): Promise<EnvVar[]> => {
    const response = await api.get<{ data: EnvVar[] }>('/env');
    return response.data.data;
  },

  create: async (data: { key: string; value: string; type: EnvVar['type']; description?: string }): Promise<EnvVar> => {
    const response = await api.post<{ data: EnvVar }>('/env', data);
    return response.data.data;
  },

  delete: async (key: string): Promise<void> => {
    await api.delete(`/env/${encodeURIComponent(key)}`);
  },

  // TAREA 3: New endpoints for .env file support
  getDotenv: async (): Promise<{ content: string }> => {
    const response = await api.get<{ content: string }>('/env/dotenv');
    return response.data;
  },

  saveDotenv: async (content: string): Promise<{ message: string; restarted: boolean }> => {
    const response = await api.put<{ message: string; restarted: boolean }>('/env/dotenv', { content });
    return response.data;
  },
};
