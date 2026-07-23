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

  bulkImport: async (content: string, commit: boolean): Promise<BulkEnvResult> => {
    try {
      const response = await api.post<unknown>('/env/bulk', { content, commit });
      const body = response.data;
      const data = (body as { data?: BulkEnvResult })?.data ?? (body as BulkEnvResult);
      return {
        lines: data?.lines ?? [],
        issues: data?.issues ?? [],
        valid: data?.valid ?? false,
        summary: data?.summary ?? '',
      };
    } catch (err) {
      // ponytail: backend errors come back without a BulkEnvResult envelope;
      // surface a synthetic report so the modal never crashes on undefined.
      const reason =
        (err as { response?: { data?: { error?: { code?: string; message?: string } } } })?.response?.data
          ?.error?.message ?? 'Import failed';
      return { lines: [], issues: [{ line: 0, reason }], valid: false, summary: reason };
    }
  },

  importFromNodeRed: async (commit: boolean): Promise<BulkEnvResult> => {
    try {
      const response = await api.post<unknown>('/env/import-from-node-red', { commit });
      const body = response.data;
      const data = (body as { data?: BulkEnvResult })?.data ?? (body as BulkEnvResult);
      return {
        lines: data?.lines ?? [],
        issues: data?.issues ?? [],
        valid: data?.valid ?? false,
        summary: data?.summary ?? '',
      };
    } catch (err) {
      const reason =
        (err as { response?: { data?: { error?: { code?: string; message?: string } } } })?.response?.data
          ?.error?.message ?? 'Import failed';
      return { lines: [], issues: [{ line: 0, reason }], valid: false, summary: reason };
    }
  },
};

export interface BulkEnvIssue {
  line: number;
  key?: string;
  reason: string;
}

export interface BulkEnvLine {
  line: number;
  key: string;
  value: string;
  type: string;
}

export interface BulkEnvResult {
  lines: BulkEnvLine[];
  issues: BulkEnvIssue[];
  valid: boolean;
  summary: string;
}
