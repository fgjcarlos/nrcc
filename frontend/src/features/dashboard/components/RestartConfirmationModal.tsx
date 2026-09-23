import { AlertTriangle, RefreshCw } from 'lucide-react';
import { createPortal } from 'react-dom';
import { useT } from '@/i18n';

interface RestartConfirmationModalProps {
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export function RestartConfirmationModal({ isOpen, onCancel, onConfirm }: RestartConfirmationModalProps) {
  const { t } = useT();
  if (!isOpen) {
    return null;
  }

  // Portaled to document.body so `position: fixed` resolves against the
  // viewport, not the surrounding <main> scroll container (which had a
  // sticky header establishing a containing block). See issue #704.
  return createPortal(
    <div className="modal-overlay" onClick={onCancel}>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="restart-confirmation-modal-title"
        className="surface-panel w-full max-w-sm border border-border p-6 shadow-glow"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex flex-col items-center gap-3 pt-2 pb-4 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-warning/10 text-warning">
            <AlertTriangle className="h-7 w-7" />
          </div>
          <div>
            <h3 id="restart-confirmation-modal-title" className="text-lg font-bold">{t('dashboard:restart.title')}</h3>
            <p className="mt-1 text-sm text-base-content/60">
              {t('dashboard:restart.description')}
            </p>
          </div>
        </div>
        <div className="flex justify-end gap-2 pt-2">
          <button onClick={onCancel} className="action-btn-secondary">
            Cancelar
          </button>
          <button onClick={onConfirm} className="action-btn-primary gap-2">
            <RefreshCw className="h-4 w-4" />
            {t('dashboard:restart.confirm')}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
