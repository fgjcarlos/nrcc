export interface FlowSummary {
  id: string;
  label: string;
  nodes: number;
  connections: number;
  disabled: boolean;
  lastModified?: string;
}

export interface FlowNode {
  id: string;
  type: string;
  name?: string;
  z: string;
  wires?: string[][];
  [key: string]: unknown;
}

export interface FlowDetail {
  id: string;
  label: string;
  nodes: FlowNode[];
  lastModified?: string;
}

export interface FlowMetrics {
  nodeCount: number;
  connectionCount: number;
  entryPoints: string[];
  exitPoints: string[];
  nodeTypes: Record<string, number>;
  disabledNodes: number;
}

export interface AnalysisResult {
  flowId: string;
  summary: string;
  pros: string[];
  cons: string[];
  suggestions: string[];
  analyzedAt: string;
}
