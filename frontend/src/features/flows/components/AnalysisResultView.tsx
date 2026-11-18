import { type AnalysisResult } from '@/features/flows';

interface AnalysisResultViewProps {
  result: AnalysisResult;
}

export function AnalysisResultView({ result }: AnalysisResultViewProps) {
  return (
    <div className="space-y-4">
      <div>
        <h3 className="mb-2 font-medium text-base-content">Summary</h3>
        <p className="text-ds-text-secondary">{result.summary}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
         <div>
           <h4 className="mb-2 font-medium text-success">Pros</h4>
           <ul className="space-y-1">
             {result.pros.map((pro, i) => (
               <li key={i} className="flex items-start gap-2 text-sm text-ds-text-secondary">
                 <span className="text-success">+</span> {pro}
               </li>
             ))}
           </ul>
         </div>

         <div>
           <h4 className="mb-2 font-medium text-error">Cons</h4>
           <ul className="space-y-1">
             {result.cons.map((con, i) => (
               <li key={i} className="flex items-start gap-2 text-sm text-ds-text-secondary">
                 <span className="text-error">-</span> {con}
               </li>
             ))}
           </ul>
         </div>

         <div>
           <h4 className="mb-2 font-medium text-info">Suggestions</h4>
           <ul className="space-y-1">
             {result.suggestions.map((sug, i) => (
               <li key={i} className="flex items-start gap-2 text-sm text-ds-text-secondary">
                 <span className="text-info">→</span> {sug}
               </li>
             ))}
           </ul>
         </div>
       </div>
    </div>
  );
}
