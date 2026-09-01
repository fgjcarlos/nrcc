import { api } from '@/shared/lib';
import type { FlowSummary, FlowDetail, FlowMetrics, AnalysisResult, FlowVersionEntry, FlowDiff, AIFlowAction, AIFlowResponse, FlowNode } from '../types';

export interface AICapabilityStatus {
  status: 'disabled' | 'incomplete' | 'testing' | 'unreachable' | 'ready';
  provider: 'offline' | 'openai';
  endpoint?: string;
  model?: string;
  reason?: string;
}

export function getAICapabilityUnavailableMessage(capability?: AICapabilityStatus): string {
  return capability?.reason || 'AI actions are unavailable until the configured provider is ready.';
}

export class AICapabilityUnavailableError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'AICapabilityUnavailableError';
  }
}

async function assertAICapabilityReady(): Promise<void> {
  let capability: AICapabilityStatus;
  try {
    capability = await flowService.getAICapability();
  } catch {
    throw new AICapabilityUnavailableError('AI actions are unavailable until the configured provider is ready.');
  }

  if (capability.status !== 'ready') {
    throw new AICapabilityUnavailableError(getAICapabilityUnavailableMessage(capability));
  }
}

export const flowService = {
  getAICapability: async (): Promise<AICapabilityStatus> => {
    const response = await api.get<{ data: AICapabilityStatus }>('/ai/status');
    return response.data.data;
  },

  getFlows: async (): Promise<{ available: boolean; flows: FlowSummary[] }> => {
    const response = await api.get<{ data: { available: boolean; flows: FlowSummary[] } }>('/flows');
    return response.data.data;
  },

  getFlowById: async (id: string): Promise<FlowDetail> => {
    const response = await api.get<{ data: FlowDetail }>(`/flows/${id}`);
    return response.data.data;
  },

  getFlowMetrics: async (id: string): Promise<FlowMetrics> => {
    const response = await api.get<{ data: FlowMetrics }>(`/flows/${id}/metrics`);
    return response.data.data;
  },

  analyzeFlow: async (flowId: string): Promise<AnalysisResult> => {
    await assertAICapabilityReady();
    const flow = await flowService.getFlowById(flowId);
    const response = await flowService.requestAIAssistance({
      action: 'audit',
      flow: { id: flow.id, label: flow.label, nodes: flow.nodes },
    });
    return {
      flowId,
      summary: response.summary,
      pros: [],
      cons: response.auditFindings ?? [],
      suggestions: response.suggestions ?? [],
      analyzedAt: new Date().toISOString(),
    };
  },

  requestAIAssistance: async (input: {
    action: AIFlowAction;
    flow: { id: string; label: string; nodes: FlowNode[] };
    prompt?: string;
  }): Promise<AIFlowResponse> => {
    await assertAICapabilityReady();
    const response = await api.post<{ data: AIFlowResponse }>('/ai/analyze/flow', input);
    return response.data.data;
  },

  getVersions: async (): Promise<FlowVersionEntry[]> => {
    const response = await api.get<{ data: FlowVersionEntry[] }>('/flows/versions');
    return response.data.data;
  },

  getVersionDiff: async (fromId: string, toId: string): Promise<FlowDiff> => {
    const response = await api.get<{ data: FlowDiff }>(`/flows/versions/${fromId}/diff/${toId}`);
    return response.data.data;
  },

  revertToVersion: async (versionId: string): Promise<void> => {
    await api.post(`/flows/versions/${versionId}/revert`);
  },

  captureSnapshot: async (): Promise<void> => {
    await api.post('/flows/versions');
  },
};
