import { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { type PatternAnalysisResult } from '@/features/patterns/services';
import { PatternCard } from '@/features/patterns/components';
import { useFlowDetailData, useFlowDetailActions } from '@/features/flows/hooks';
import { MetricCard } from './MetricCard';
import { AnalysisResultView } from './AnalysisResultView';
import {
  ArrowLeft,
  Activity,
  AlertTriangle,
  Lightbulb,
  Loader2,
  GitBranch,
  CheckSquare,
  Square,
} from 'lucide-react';
import { toast } from 'sonner';
import { cn } from '@/shared/lib';

export function FlowDetailView() {
  const { id } = useParams<{ id: string }>();

  // Hooks for data and actions
  const { flow, metrics, allFlows, isLoading, flowLoading } = useFlowDetailData({
    flowId: id,
  });
  const { analyzeFlowMutation, detectPatternsMutation } = useFlowDetailActions();

  // UI State
  const [showRawJson, setShowRawJson] = useState(false);
  const [showPatternSelector, setShowPatternSelector] = useState(false);
  const [selectedPatternFlows, setSelectedPatternFlows] = useState<Set<string>>(
    new Set()
  );
  const [patternResult, setPatternResult] = useState<PatternAnalysisResult | null>(
    null
  );

  // Handlers
  const togglePatternFlow = (flowId: string) => {
    const next = new Set(selectedPatternFlows);
    if (next.has(flowId)) {
      next.delete(flowId);
    } else {
      next.add(flowId);
    }
    setSelectedPatternFlows(next);
  };

  const handleDetectPatterns = () => {
    if (selectedPatternFlows.size < 1) {
      toast.error('Select at least 1 flow');
      return;
    }
    detectPatternsMutation.mutate(Array.from(selectedPatternFlows), {
      onSuccess: (data) => {
        setPatternResult(data);
        setShowPatternSelector(false);
      },
    });
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="p-6">
        <div className="surface-card animate-pulse space-y-4 p-6">
          <div className="h-8 w-1/4 rounded skeleton-dark"></div>
          <div className="h-32 rounded skeleton-dark"></div>
        </div>
      </div>
    );
  }

  // Not found state
  if (!flow) {
    return (
      <div className="p-6">
        <p className="text-base-content/60">Flow not found</p>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link
          to="/flows"
          className="action-btn-ghost p-2 text-base-content/70 hover:text-base-content"
        >
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <h1 className="text-2xl font-bold text-base-content">{flow.label}</h1>
      </div>

      {/* Metrics */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <MetricCard
          icon={Activity}
          label="Nodes"
          value={metrics?.nodeCount || 0}
        />
        <MetricCard
          icon={Activity}
          label="Connections"
          value={metrics?.connectionCount || 0}
        />
        <MetricCard
          icon={AlertTriangle}
          label="Disabled"
          value={metrics?.disabledNodes || 0}
          warning={metrics?.disabledNodes ? metrics.disabledNodes > 0 : false}
        />
        <MetricCard
          icon={Lightbulb}
          label="Node Types"
          value={Object.keys(metrics?.nodeTypes || {}).length}
        />
      </div>

      {/* AI Analysis */}
      <div className="surface-card space-y-4 p-6">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-base-content">AI Analysis</h2>
          <div className="flex items-center gap-2">
            <button
              onClick={() => analyzeFlowMutation.mutate(id!)}
              disabled={analyzeFlowMutation.isPending}
              className="action-btn-primary"
            >
              {analyzeFlowMutation.isPending && (
                <Loader2 className="w-4 h-4 animate-spin" />
              )}
              Analyze Flow
            </button>
          </div>
        </div>

        {/* Analyze Flow Result */}
        {analyzeFlowMutation.isPending ? (
          <div className="flex items-center gap-2 text-base-content/60">
            <Loader2 className="w-5 h-5 animate-spin" />
            Analyzing flow...
          </div>
        ) : analyzeFlowMutation.data ? (
          <AnalysisResultView result={analyzeFlowMutation.data} />
        ) : (
          <p className="text-base-content/60">
            Click "Analyze Flow" to get insights about this flow.
          </p>
        )}

        {/* Divider */}
        {(analyzeFlowMutation.data || analyzeFlowMutation.isPending) && (
          <div className="border-t ghost-divider" />
        )}

        {/* Detect Patterns Section */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <GitBranch className="w-4 h-4 text-base-content/60" />
              <span className="text-sm text-base-content/60">
                Detect Reusable Patterns
              </span>
            </div>
            <button
              onClick={() => setShowPatternSelector(!showPatternSelector)}
              className="action-btn-ghost text-sm"
            >
              <GitBranch className="w-4 h-4" />
              {showPatternSelector ? 'Hide' : 'Detect Patterns'}
            </button>
          </div>

          {/* Flow Selector */}
          {showPatternSelector && (
            <div className="surface-panel space-y-4 p-4">
              <p className="text-sm text-base-content/60">
                Select flows to compare and detect reusable patterns. The
                current flow is automatically selected.
              </p>

              <div className="space-y-2 max-h-60 overflow-y-auto">
                {allFlows?.flows.map((f) => (
                  <div
                    key={f.id}
                    className={cn(
                      'flex cursor-pointer items-center gap-3 rounded-xl p-2 transition-colors',
                      'hover:skeleton-dark',
                      selectedPatternFlows.has(f.id) && 'bg-primary/10'
                    )}
                    onClick={() => togglePatternFlow(f.id)}
                  >
                    <button
                      className="shrink-0 text-base-content/60 hover:text-primary"
                      aria-label={
                        selectedPatternFlows.has(f.id)
                          ? 'Deseleccionar'
                          : 'Seleccionar'
                      }
                    >
                      {selectedPatternFlows.has(f.id) ? (
                        <CheckSquare className="w-4 h-4 text-primary" />
                      ) : (
                        <Square className="w-4 h-4" />
                      )}
                    </button>
                    <div className="flex-1 min-w-0">
                      <span className="text-sm font-medium text-base-content">
                        {f.label}
                      </span>
                    </div>
                    <span className="text-xs text-base-content/60">
                      {f.nodes} nodes
                    </span>
                    {f.id === id && (
                      <span className="rounded px-1.5 py-0.5 text-xs bg-primary/20 text-primary">
                        Current
                      </span>
                    )}
                  </div>
                ))}
              </div>

              <div className="flex items-center justify-between border-t ghost-divider pt-2">
                <span className="text-sm text-base-content/60">
                  {selectedPatternFlows.size} flow
                  {selectedPatternFlows.size !== 1 ? 's' : ''} selected
                </span>
                <button
                  onClick={handleDetectPatterns}
                  disabled={
                    detectPatternsMutation.isPending ||
                    selectedPatternFlows.size < 1
                  }
                  className="action-btn-primary text-sm"
                >
                  {detectPatternsMutation.isPending && (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  )}
                  {detectPatternsMutation.isPending ? 'Analyzing...' : 'Detect Patterns'}
                </button>
              </div>
            </div>
          )}

          {/* Pattern Detection Loading */}
          {detectPatternsMutation.isPending && (
            <div className="flex items-center gap-2 text-base-content/60">
              <Loader2 className="w-5 h-5 animate-spin" />
              Detecting patterns across {selectedPatternFlows.size} flow
              {selectedPatternFlows.size !== 1 ? 's' : ''}...
            </div>
          )}

          {/* No Patterns Found */}
           {patternResult && patternResult.patterns.length === 0 && (
             <div className="surface-panel border border-warning/20 p-4">
               <p className="text-sm text-warning">
                {patternResult.message ||
                  "No patterns detected in the selected flow(s). Try selecting flows with similar structures or data transformations."}
              </p>
            </div>
          )}

          {/* Pattern Results */}
          {patternResult && patternResult.patterns.length > 0 && (
            <div className="space-y-4 border-t ghost-divider pt-4">
              <h3 className="font-medium text-base-content">
                Detected Patterns ({patternResult.patterns.length})
              </h3>
              <div className="grid gap-4">
                {patternResult.patterns.map((pattern) => (
                  <PatternCard
                    key={pattern.id}
                    pattern={pattern}
                    analysisId={patternResult.patternId}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Node Types */}
      <div className="surface-card p-6">
        <h2 className="mb-4 text-lg font-semibold text-base-content">
          Node Types
        </h2>
        <div className="flex flex-wrap gap-2">
          {metrics?.nodeTypes &&
            Object.entries(metrics.nodeTypes).map(([type, count]) => (
              <span
                key={type}
                className="glass-panel rounded-full border border-border px-3 py-1 text-sm text-base-content"
              >
                {type} ({count})
              </span>
            ))}
        </div>
      </div>

      {/* Raw JSON */}
      <div className="surface-card p-6">
        <button
          onClick={() => setShowRawJson(!showRawJson)}
          className="text-sm text-base-content/60 transition-colors hover:text-base-content"
        >
          {showRawJson ? 'Hide' : 'Show'} raw JSON
        </button>
        {showRawJson && (
          <pre className="mt-4 overflow-x-auto code-block-bg text-xs text-base-content/70">
            {JSON.stringify(flow.nodes, null, 2)}
          </pre>
        )}
      </div>
    </div>
  );
}
