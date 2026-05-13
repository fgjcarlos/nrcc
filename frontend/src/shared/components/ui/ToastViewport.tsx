import { createPortal } from 'react-dom';
import { X, CheckCircle, AlertTriangle, XCircle, Info } from 'lucide-react';
import { useToastStore, type Toast, type ToastTone } from '@/shared/hooks/useToasts';
import { cn } from '@/shared/lib';

// ── Tone → daisyUI alert class + icon ────────────────────────────────────────
const toneMap: Record<ToastTone, { cls: string; Icon: React.ElementType }> = {
  success: { cls: 'alert-success', Icon: CheckCircle },
  error:   { cls: 'alert-error',   Icon: XCircle     },
  warning: { cls: 'alert-warning', Icon: AlertTriangle },
  info:    { cls: 'alert-info',    Icon: Info         },
};

function ToastItem({ toast }: { toast: Toast }) {
  const dismiss = useToastStore((s) => s.dismiss);
  const { cls, Icon } = toneMap[toast.tone];

  return (
    <div
      role="alert"
      className={cn(
        'alert shadow-lg max-w-sm w-full flex items-start gap-3 pr-3',
        cls,
      )}
    >
      <Icon className="w-5 h-5 shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0">
        <p className="font-semibold text-sm leading-snug">{toast.title}</p>
        {toast.message && (
          <p className="text-xs opacity-80 mt-0.5 break-words">{toast.message}</p>
        )}
      </div>
      <button
        onClick={() => dismiss(toast.id)}
        className="btn btn-ghost btn-xs btn-circle shrink-0 opacity-60 hover:opacity-100"
        aria-label="Cerrar"
      >
        <X className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}

/**
 * Renders all active toasts in a fixed overlay (bottom-end).
 * Mount once, anywhere above the route tree (e.g. in App.tsx or Layout).
 */
export function ToastViewport() {
  const toasts = useToastStore((s) => s.toasts);

  if (toasts.length === 0) return null;

  return createPortal(
    <div className="toast toast-end toast-bottom z-[9999] gap-2 p-4">
      {toasts.map((t) => (
        <ToastItem key={t.id} toast={t} />
      ))}
    </div>,
    document.body,
  );
}
