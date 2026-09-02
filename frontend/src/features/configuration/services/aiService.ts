import { api } from '@/shared/lib';

export type AIProvider = 'offline' | 'openai';

export interface AIConfig {
  enabled: boolean;
  provider: AIProvider;
  endpoint?: string;
  model?: string;
  apiKeyConfigured: boolean;
}

export interface AIConfigInput {
  enabled: boolean;
  provider: AIProvider;
  endpoint?: string;
  model?: string;
  apiKey?: string;
}

export interface AIProviderStatus {
  status: 'disabled' | 'incomplete' | 'testing' | 'unreachable' | 'ready';
  provider: AIProvider;
  endpoint?: string;
  model?: string;
  reason?: string;
}

export const aiService = {
  async getConfig(): Promise<AIConfig> {
    const response = await api.get<{ data: AIConfig }>('/ai/config');
    return response.data.data;
  },

  async saveConfig(config: AIConfigInput): Promise<AIConfig> {
    const response = await api.put<{ data: AIConfig }>('/ai/config', config);
    return response.data.data;
  },

  async testConfig(): Promise<AIProviderStatus> {
    const response = await api.post<{ data: AIProviderStatus }>('/ai/config/test');
    return response.data.data;
  },

  async getStatus(): Promise<AIProviderStatus> {
    const response = await api.get<{ data: AIProviderStatus }>('/ai/status');
    return response.data.data;
  },
};
