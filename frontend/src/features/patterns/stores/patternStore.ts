import { create } from 'zustand';

export interface NodeProperty {
  name: string;
  type: 'str' | 'num' | 'bool' | 'json' | 'msg';
  default?: string | number | boolean;
  description: string;
}

export interface DetectedPattern {
  id: string;
  name: string;
  description: string;
  frequency: number;
  flows: string[];
  nodeSuggestion: {
    name: string;
    category: string;
    inputs: number;
    outputs: number;
    properties: NodeProperty[];
  };
}

export interface PatternAnalysisResult {
  patternId: string;
  patterns: DetectedPattern[];
  analyzedAt: string;
  flowCount: number;
  message?: string;
}

interface PatternState {
  // Selection state
  selectedFlowIds: Set<string>;
  
  // Analysis state
  analyzing: boolean;
  lastAnalysis: PatternAnalysisResult | null;
  error: string | null;
  
  // Actions
  toggleFlow: (flowId: string) => void;
  selectAll: (flowIds: string[]) => void;
  clearSelection: () => void;
  setAnalyzing: (value: boolean) => void;
  setLastAnalysis: (result: PatternAnalysisResult) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

export const usePatternStore = create<PatternState>((set) => ({
  // Initial state
  selectedFlowIds: new Set(),
  analyzing: false,
  lastAnalysis: null,
  error: null,

  // Actions
  toggleFlow: (flowId) =>
    set((state) => {
      const next = new Set(state.selectedFlowIds);
      if (next.has(flowId)) {
        next.delete(flowId);
      } else {
        next.add(flowId);
      }
      return { selectedFlowIds: next };
    }),

  selectAll: (flowIds) =>
    set({ selectedFlowIds: new Set(flowIds) }),

  clearSelection: () =>
    set({ selectedFlowIds: new Set() }),

  setAnalyzing: (value) =>
    set({ analyzing: value }),

  setLastAnalysis: (result) =>
    set({ lastAnalysis: result, error: null }),

  setError: (error) =>
    set({ error }),

  reset: () =>
    set({
      selectedFlowIds: new Set(),
      analyzing: false,
      lastAnalysis: null,
      error: null,
    }),
}));
