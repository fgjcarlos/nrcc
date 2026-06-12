import { api } from '@/shared/lib';
import { unwrap, type Schemas } from '@/shared/api';

/**
 * Environment variable shape, sourced from the OpenAPI spec (`EnvVar`) instead
 * of a hand-written duplicate. If the spec changes, this — and every consumer —
 * breaks at compile time. Re-exported for components that render env vars.
 */
export type EnvVar = Schemas['EnvVar'];

export const envService = {
  getAll: async (): Promise<EnvVar[]> => {
    const response = await api.get<Schemas['SuccessEnvelope_EnvVarList']>('/env');
    return unwrap(response.data);
  },

  create: async (data: Schemas['SetEnvRequest']): Promise<Schemas['EnvSetResult']> => {
    const response = await api.post<Schemas['SuccessEnvelope_EnvSetResult']>('/env', data);
    return unwrap(response.data);
  },

  delete: async (key: string): Promise<void> => {
    await api.delete(`/env/${encodeURIComponent(key)}`);
  },

  // .env file support
  getDotenv: async (): Promise<Schemas['DotenvContent']> => {
    const response = await api.get<Schemas['SuccessEnvelope_DotenvContent']>('/env/dotenv');
    return unwrap(response.data);
  },

  saveDotenv: async (content: string): Promise<Schemas['EnvSetResult']> => {
    const response = await api.put<Schemas['SuccessEnvelope_EnvSetResult']>('/env/dotenv', { content });
    return unwrap(response.data);
  },
};
