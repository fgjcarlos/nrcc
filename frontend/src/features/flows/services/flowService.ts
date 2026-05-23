import { api } from '@/shared/lib';
import type { FlowSummary, FlowDetail, FlowMetrics, AnalysisResult, FlowVersionEntry, FlowDiff } from '../types';

export const flowService = {
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
    const response = await api.post<{ data: AnalysisResult }>(`/flows/${flowId}/analyze`);
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
