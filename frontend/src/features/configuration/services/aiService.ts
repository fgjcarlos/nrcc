import { api } from '@/shared/lib';

export type AIProvider = 'offline' | 'openai' | 'anthropic' | 'ollama';

export interface AIConfig {
  enabled: boolean;
  provider: AIProvider;
  endpoint?: string;
  model?: string;
  apiKeyConfigured: boolean;
  allowPrivateHTTP?: boolean;
  timeoutSeconds?: number;
}

export interface AIConfigInput {
  enabled: boolean;
  provider: AIProvider;
  endpoint?: string;
  model?: string;
  apiKey?: string;
  allowPrivateHTTP?: boolean;
  timeoutSeconds?: number;
}

export interface AIProviderStatus {
  status: 'disabled' | 'incomplete' | 'testing' | 'unreachable' | 'ready';
  provider: AIProvider;
  endpoint?: string;
  model?: string;
  reason?: string;
}

export type FlowAIAction = 'explain' | 'suggest' | 'audit' | 'generate';
export interface FlowAIResponse {
  summary: string;
  suggestions?: string[];
  auditFindings?: string[];
  candidateFlow?: { nodes: Record<string, unknown>[] };
  previewToken?: string;
  request?: { endpoint?: string; model?: string; messages?: unknown[]; body?: unknown };
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

  async analyzeFlow(input: { action: FlowAIAction; flow: unknown; prompt?: string; previewOnly?: boolean; previewToken?: string }): Promise<FlowAIResponse> {
    const response = await api.post<{ data: FlowAIResponse }>('/ai/analyze/flow', input);
    return response.data.data;
  },

  async getStatus(): Promise<AIProviderStatus> {
    const response = await api.get<{ data: AIProviderStatus }>('/ai/status');
    return response.data.data;
  },
};
