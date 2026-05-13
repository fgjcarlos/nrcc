import { useState } from 'react';
import { type FlowSummary, type AnalysisResult } from '@/features/flows';
import { Link } from 'react-router-dom';
import { cn } from '@/shared/lib';
import {
  Activity,
  AlertCircle,
  Loader2,
  Sparkles,
  CheckSquare,
  Square,
  ChevronDown,
  ChevronUp,
  ThumbsUp,
  ThumbsDown,
  Lightbulb,
  AlertTriangle,
} from 'lucide-react';
import { useFlowsData } from '@/features/flows/hooks/useFlowsData';
import { useFlowsActions } from '@/features/flows/hooks/useFlowsActions';

export function FlowsView() {
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [analysisResults, setAnalysisResults] = useState<Record<string, AnalysisResult>>({});
  const [analyzing, setAnalyzing] = useState(false);
  const [expandedResults, setExpandedResults] = useState<Set<string>>(new Set());

  const { flows, available, isLoading, error } = useFlowsData();
  const { analyzeFlows } = useFlowsActions();

  const toggleSelect = (id: string) => {
    setSelected(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (flows.length === 0) return;
    if (selected.size === flows.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(flows.map(f => f.id)));
    }
  };

  const toggleResultExpanded = (id: string) => {
    setExpandedResults(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const handleAnalyze = async () => {
    if (selected.size === 0) return;
    setAnalyzing(true);
    const ids = Array.from(selected);
    const results = await analyzeFlows(ids);
    setAnalysisResults(prev => ({ ...prev, ...results }));
    setExpandedResults(new Set(ids.filter(id => results[id])));
    setAnalyzing(false);
  };

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">AI pipeline</p>
          <h1 className="text-2xl font-bold text-base-content">Flows</h1>
        </div>
        <div className="space-y-3 animate-pulse">
          {[1, 2, 3].map(i => (
            <div key={i} className="h-24 rounded-2xl skeleton-dark" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">AI pipeline</p>
          <h1 className="text-2xl font-bold text-base-content">Flows</h1>
        </div>
        <div className="mt-4 rounded-2xl border border-error/20 bg-error/10 p-4 shadow-glow">
          <p className="text-error">Error loading flows: {error.message}</p>
        </div>
      </div>
    );
  }

  if (!available) {
    return (
      <div className="p-6">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">AI pipeline</p>
          <h1 className="text-2xl font-bold text-base-content">Flows</h1>
        </div>
        <div className="mt-4 flex items-start gap-3 rounded-2xl border border-warning/20 bg-warning/10 p-4 shadow-glow">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
          <div>
            <p className="font-medium text-base-content">Node-RED no está disponible</p>
            <p className="mt-1 text-sm text-base-content/70">
              Verificá que el contenedor de Node-RED esté corriendo. Podés reiniciarlo desde la sección Runtime.
            </p>
          </div>
        </div>
      </div>
    );
  }

  const allSelected = flows.length > 0 && selected.size === flows.length;

  return (
    <div className="space-y-6 p-6 pb-32">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div>
            <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">AI pipeline</p>
            <h1 className="text-2xl font-bold text-base-content">Flows</h1>
          </div>
          <span className="text-sm text-base-content/60">
            {flows.length} flow{flows.length !== 1 ? 's' : ''}
          </span>
        </div>
        {flows.length > 0 && (
          <button
            onClick={toggleSelectAll}
            className="flex items-center gap-2 text-sm text-base-content/60 transition-colors hover:text-base-content"
          >
            {allSelected
              ? <CheckSquare className="w-4 h-4" />
              : <Square className="w-4 h-4" />}
            {allSelected ? 'Deseleccionar todos' : 'Seleccionar todos'}
          </button>
        )}
      </div>

      {flows.length === 0 ? (
        <div className="py-12 text-center">
          <Activity className="mx-auto mb-4 h-12 w-12 text-base-content/40" />
          <p className="text-base-content/60">No se encontraron flows</p>
        </div>
      ) : (
        <div className="grid gap-4">
          {flows.map(flow => (
            <div key={flow.id} className="space-y-0">
              <FlowCard
                flow={flow}
                selected={selected.has(flow.id)}
                onToggleSelect={() => toggleSelect(flow.id)}
              />
              {analysisResults[flow.id] && (
                <AnalysisPanel
                  result={analysisResults[flow.id]}
                  expanded={expandedResults.has(flow.id)}
                  onToggle={() => toggleResultExpanded(flow.id)}
                />
              )}
            </div>
          ))}
        </div>
      )}

      {/* Barra flotante de análisis IA */}
      {selected.size > 0 && (
        <div className="fixed z-50 -translate-x-1/2 bottom-6 left-1/2">
          <div className="glass-panel flex items-center gap-4 rounded-full border border-border px-6 py-3 shadow-glow">
            <span className="text-sm text-base-content/65">
              {selected.size} flow{selected.size !== 1 ? 's' : ''} seleccionado{selected.size !== 1 ? 's' : ''}
            </span>
            <button
              onClick={handleAnalyze}
              disabled={analyzing}
              className="flex items-center gap-2 rounded-full bg-primary px-4 py-1.5 text-sm font-medium text-primary-content transition-colors hover:bg-primary/90 disabled:opacity-50"
            >
              {analyzing
                ? <Loader2 className="w-4 h-4 animate-spin" />
                : <Sparkles className="w-4 h-4" />}
              {analyzing ? 'Analizando...' : 'Analizar con IA'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function FlowCard({
  flow,
  selected,
  onToggleSelect,
}: {
  flow: FlowSummary;
  selected: boolean;
  onToggleSelect: () => void;
}) {
  return (
    <div
      className={cn(
        'surface-card flex items-start gap-3 border p-4 transition-colors',
        selected ? 'border-primary/40 bg-primary/8' : 'border-border hover:border-primary/20',
      )}
    >
      {/* Checkbox */}
      <button
        onClick={onToggleSelect}
        className="mt-1 shrink-0 text-base-content/60 transition-colors hover:text-primary"
        aria-label={selected ? 'Deseleccionar flow' : 'Seleccionar flow'}
      >
        {selected
          ? <CheckSquare className="w-5 h-5 text-primary" />
          : <Square className="w-5 h-5" />}
      </button>

      {/* Icono de estado */}
      <div className={cn(
        'p-2 rounded-2xl shrink-0',
        flow.disabled ? 'bg-warning/10' : 'bg-success/10',
      )}>
        {flow.disabled
          ? <AlertCircle className="w-5 h-5 text-warning" />
          : <Activity className="w-5 h-5 text-success" />}
      </div>

      {/* Info + link */}
      <div className="flex-1 min-w-0">
        <Link
          to={`/flows/${flow.id}`}
          className="font-medium text-base-content transition-colors hover:text-primary"
        >
          {flow.label}
        </Link>
        <div className="mt-1 flex items-center gap-4 text-sm text-base-content/65">
          <span>{flow.nodes} nodos</span>
          <span>{flow.connections} conexiones</span>
          {flow.disabled && (
            <span className="text-warning">Deshabilitado</span>
          )}
        </div>
      </div>
    </div>
  );
}

function AnalysisPanel({
  result,
  expanded,
  onToggle,
}: {
  result: AnalysisResult;
  expanded: boolean;
  onToggle: () => void;
}) {
  return (
    <div className="overflow-hidden rounded-b-2xl border border-t-0 border-primary/25 bg-primary/8">
      <button
        onClick={onToggle}
        className="flex w-full items-center justify-between px-4 py-2 text-sm text-primary transition-colors hover:bg-primary/10"
      >
        <span className="flex items-center gap-2 font-medium">
          <Sparkles className="w-4 h-4" />
          Análisis IA
        </span>
        {expanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
      </button>

      {expanded && (
        <div className="space-y-4 px-4 pb-4">
          <p className="text-sm text-base-content">{result.summary}</p>

          {result.pros.length > 0 && (
            <div>
              <div className="mb-1.5 flex items-center gap-1.5 text-xs font-semibold text-success-content">
                <ThumbsUp className="w-3.5 h-3.5" />
                Puntos positivos
              </div>
              <ul className="space-y-1">
                {result.pros.map((item, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-base-content/65">
                    <span className="mt-0.5 text-success">•</span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {result.cons.length > 0 && (
            <div>
              <div className="mb-1.5 flex items-center gap-1.5 text-xs font-semibold text-error-content">
                <ThumbsDown className="w-3.5 h-3.5" />
                Puntos a mejorar
              </div>
              <ul className="space-y-1">
                {result.cons.map((item, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-base-content/65">
                    <span className="mt-0.5 text-error">•</span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {result.suggestions.length > 0 && (
            <div>
              <div className="mb-1.5 flex items-center gap-1.5 text-xs font-semibold text-info-content">
                <Lightbulb className="w-3.5 h-3.5" />
                Sugerencias
              </div>
              <ul className="space-y-1">
                {result.suggestions.map((item, i) => (
                  <li key={i} className="flex items-start gap-2 text-sm text-base-content/65">
                    <span className="mt-0.5 text-info">•</span>
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          )}

          <p className="text-xs text-base-content/50">
            Analizado el {new Date(result.analyzedAt).toLocaleString('es-AR')}
          </p>
        </div>
      )}
    </div>
  );
}
