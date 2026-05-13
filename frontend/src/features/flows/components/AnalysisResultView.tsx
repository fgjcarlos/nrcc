import { type AnalysisResult } from '@/features/flows';

interface AnalysisResultViewProps {
  result: AnalysisResult;
}

export function AnalysisResultView({ result }: AnalysisResultViewProps) {
  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-2 font-medium text-base-content">Summary</h3>
        <p className="text-base-content/60">{result.summary}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <h4 className="mb-2 font-medium text-emerald-300">Pros</h4>
          <ul className="space-y-1">
            {result.pros.map((pro, i) => (
              <li key={i} className="flex items-start gap-2 text-sm text-base-content/60">
                <span className="text-emerald-400">+</span> {pro}
              </li>
            ))}
          </ul>
        </div>

        <div>
          <h4 className="mb-2 font-medium text-rose-300">Cons</h4>
          <ul className="space-y-1">
            {result.cons.map((con, i) => (
              <li key={i} className="flex items-start gap-2 text-sm text-base-content/60">
                <span className="text-rose-400">-</span> {con}
              </li>
            ))}
          </ul>
        </div>

        <div>
          <h4 className="mb-2 font-medium text-sky-300">Suggestions</h4>
          <ul className="space-y-1">
            {result.suggestions.map((sug, i) => (
              <li key={i} className="flex items-start gap-2 text-sm text-base-content/60">
                <span className="text-sky-400">→</span> {sug}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}
