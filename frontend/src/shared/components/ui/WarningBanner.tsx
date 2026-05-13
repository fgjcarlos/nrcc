import { useState } from 'react';
import { AlertTriangle, X } from 'lucide-react';
import { cn } from '@/shared/lib/utils';

interface WarningBannerProps {
  message: string;
  className?: string;
}

export function WarningBanner({ message, className }: WarningBannerProps) {
  const [dismissed, setDismissed] = useState(false);

  if (dismissed) return null;

  return (
    <div
      role="alert"
      className={cn(
        'flex items-center justify-between gap-2 rounded-2xl border border-warning/20 bg-warning/10 px-4 py-3 text-warning-content',
        className
      )}
    >
      <div className="flex items-center gap-2">
        <AlertTriangle className="h-4 w-4 text-warning" />
        <span className="text-sm text-base-content">{message}</span>
      </div>
      <button
        onClick={() => setDismissed(true)}
        className="btn btn-ghost btn-sm flex items-center gap-1 text-base-content/70"
      >
        <X className="h-3 w-3" />
        Dismiss
      </button>
    </div>
  );
}
