import { AlertTriangle, RefreshCw } from 'lucide-react';

interface RestartConfirmationModalProps {
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export function RestartConfirmationModal({ isOpen, onCancel, onConfirm }: RestartConfirmationModalProps) {
  if (!isOpen) {
    return null;
  }

  return (
    <div className="modal modal-open">
      <div className="modal-box max-w-sm">
        <div className="flex flex-col items-center gap-3 pt-2 pb-4 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-warning/10 text-warning">
            <AlertTriangle className="h-7 w-7" />
          </div>
          <div>
            <h3 className="text-lg font-bold">¿Reiniciar Node-RED?</h3>
            <p className="mt-1 text-sm text-base-content/60">
              Node-RED se detendrá y volverá a arrancar. Los flujos activos se interrumpirán brevemente.
            </p>
          </div>
        </div>
        <div className="modal-action mt-0">
          <button onClick={onCancel} className="btn btn-ghost flex-1">
            Cancelar
          </button>
          <button onClick={onConfirm} className="btn btn-warning flex-1 gap-2">
            <RefreshCw className="h-4 w-4" />
            Sí, reiniciar
          </button>
        </div>
      </div>
      <div className="modal-backdrop" onClick={onCancel} />
    </div>
  );
}
