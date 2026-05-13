import { useState } from 'react';
import { Download, FileText, Layers, ArrowRight } from 'lucide-react';
import { cn } from '@/shared/lib';
import { patternService } from '../services';
import type { DetectedPattern } from '../stores/patternStore';
import { toast } from 'sonner';

interface PatternCardProps {
  pattern: DetectedPattern;
  analysisId: string;
}

export function PatternCard({ pattern, analysisId }: PatternCardProps) {
  const [downloading, setDownloading] = useState(false);

  const handleDownload = async () => {
    setDownloading(true);
    try {
      await patternService.downloadReadme(analysisId, pattern.id);
      toast.success('README downloaded successfully');
    } catch (error) {
      toast.error('Failed to download README');
      console.error('Download error:', error);
    } finally {
      setDownloading(false);
    }
  };

  return (
    <div className="surface-card space-y-4 border border-border p-4">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <h3 className="truncate font-semibold text-base-content">
            {pattern.name}
          </h3>
          <p className="mt-1 line-clamp-2 text-sm text-base-content/70">
            {pattern.description}
          </p>
        </div>
        <button
          onClick={handleDownload}
          disabled={downloading}
          className={cn(
            'flex items-center gap-2 px-3 py-1.5 text-sm rounded-md transition-colors shrink-0',
            'bg-primary text-primary-foreground hover:bg-primary/90',
            'disabled:opacity-50 disabled:cursor-not-allowed'
          )}
        >
          <Download className="w-4 h-4" />
          {downloading ? 'Downloading...' : 'README'}
        </button>
      </div>

      {/* Stats */}
      <div className="flex flex-wrap gap-4 text-sm text-base-content/65">
        <div className="flex items-center gap-1.5">
          <Layers className="w-4 h-4" />
          <span>Found in {pattern.frequency} flow{pattern.frequency !== 1 ? 's' : ''}</span>
        </div>
        <div className="flex items-center gap-1.5">
          <ArrowRight className="w-4 h-4" />
          <span>In: {pattern.nodeSuggestion.inputs} / Out: {pattern.nodeSuggestion.outputs}</span>
        </div>
      </div>

      {/* Node Suggestion */}
      <div className="pt-3 border-t border-border">
        <div className="mb-2 flex items-center gap-2 text-xs text-base-content/60">
          <FileText className="w-3.5 h-3.5" />
          <span>Suggested Node: <code className="rounded bg-base-300/70 px-1 py-0.5 text-base-content">{pattern.nodeSuggestion.name}</code></span>
        </div>
        
        {pattern.nodeSuggestion.properties.length > 0 && (
          <div className="text-xs space-y-1">
            <span className="font-medium text-base-content/60">Properties:</span>
            <ul className="grid gap-1">
              {pattern.nodeSuggestion.properties.map((prop) => (
                <li key={prop.name} className="flex items-center gap-2 text-base-content/65">
                  <code className="rounded bg-base-300/70 px-1 py-0.5 text-base-content">
                    {prop.name}
                  </code>
                  <span className="text-base-content/60">— {prop.description}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
