import { api } from 'shared/lib/api';
import { authService } from '@/features/auth/services/authService';
import type { DetectedPattern, PatternAnalysisResult, NodeProperty } from '../stores/patternStore';

export type { DetectedPattern, PatternAnalysisResult, NodeProperty };

export interface PatternAnalysisRequest {
  flowIds: string[];
  provider?: 'openai' | 'anthropic' | 'gemini' | 'ollama' | 'openrouter';
}

export interface ReadmeResponse {
  readme: string;
  patternName: string;
}

export const patternService = {
  /**
   * Analyze multiple flows to detect reusable patterns
   */
  analyzePatterns: async (request: PatternAnalysisRequest): Promise<PatternAnalysisResult> => {
    const response = await api.post<{ data: PatternAnalysisResult }>('/ai/analyze/patterns', request, {
      timeout: 60000,
    });
    return response.data.data;
  },

  /**
   * Get README content for a specific pattern
   */
  getReadme: async (analysisId: string, patternId: string): Promise<ReadmeResponse> => {
    const response = await api.get<{ data: ReadmeResponse }>(
      `/ai/patterns/${analysisId}/readme?patternId=${patternId}`
    );
    return response.data.data;
  },

  /**
   * Download README as a file
   */
  downloadReadme: async (analysisId: string, patternId: string): Promise<void> => {
    const token = authService.getToken();
    const response = await fetch(`/api/ai/patterns/${analysisId}/download?patternId=${patternId}`, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error('Failed to download README');
    }

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    
    // Extract filename from Content-Disposition header
    const contentDisposition = response.headers.get('Content-Disposition');
    const filenameMatch = contentDisposition?.match(/filename="?([^"]+)"?/);
    a.download = filenameMatch ? filenameMatch[1] : 'pattern-readme.md';
    
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
  },
};
