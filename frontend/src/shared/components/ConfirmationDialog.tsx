import { useState, useEffect, useRef, useMemo } from 'react';
import { AlertTriangle, X } from 'lucide-react';
import { UI_COPY } from '@/shared/constants/uiCopy';

export type ConfirmationVariant = 'danger' | 'warning' | 'default';

interface ConfirmationDialogProps {
  isOpen: boolean;
  title: string;
  description: string;
  confirmText?: string;
  acknowledgement?: string;
  variant?: ConfirmationVariant;
  isPending?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmationDialog({
  isOpen,
  title,
  description,
  confirmText = '',
  acknowledgement,
  variant = 'default',
  isPending = false,
  onConfirm,
  onCancel,
}: ConfirmationDialogProps) {
  const [inputValue, setInputValue] = useState('');
  const [acknowledged, setAcknowledged] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Memoize canConfirm to prevent stale closures in useEffect dependency array.
  // The Confirm button is enabled only when every gate has been satisfied:
  //   - if `confirmText` is provided, the user must type it verbatim
  //   - if `acknowledgement` is provided, the user must tick the checkbox
  const canConfirm = useMemo(
    () => () => {
      if (confirmText && inputValue !== confirmText) return false;
      if (acknowledgement && !acknowledged) return false;
      return true;
    },
    [confirmText, inputValue, acknowledgement, acknowledged]
  );

  // Reset state when the dialog opens, then focus the input.
  useEffect(() => {
    if (isOpen) {
      setInputValue('');
      setAcknowledged(false);
      if (confirmText) {
        setTimeout(() => inputRef.current?.focus(), 100);
      }
    }
  }, [isOpen, confirmText]);

  // Handle escape / enter
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen && !isPending) {
        onCancel();
      }
      if (e.key === 'Enter' && isOpen && !isPending && canConfirm()) {
        onConfirm();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, isPending, onCancel, onConfirm, canConfirm]);

  if (!isOpen) return null;

  const getVariantStyles = () => {
    switch (variant) {
      case 'danger':
        return {
          button: 'bg-error text-error-content hover:opacity-90',
          icon: 'text-error',
          border: 'border-border',
        };
      case 'warning':
        return {
          button: 'bg-warning text-warning-content hover:opacity-90',
          icon: 'text-warning',
          border: 'border-border',
        };
      default:
        return {
          button: 'bg-primary hover:bg-primary/90 text-primary-foreground',
          icon: 'text-primary',
          border: 'border-border',
        };
    }
  };

  const styles = getVariantStyles();

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 modal-overlay"
        onClick={isPending ? undefined : onCancel}
      />

      {/* Dialog */}
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirmation-dialog-title"
        className={`relative surface-panel border ${styles.border} w-full max-w-md mx-4 overflow-hidden shadow-glow`}
      >
        {/* Header */}
        <div className="flex items-start justify-between border-b ghost-divider modal-inner p-6">
          <div className="flex items-center gap-3">
            <div className={`rounded-2xl bg-base-200/70 p-2 ${styles.icon}`}>
              <AlertTriangle className="w-5 h-5" />
            </div>
            <h3 id="confirmation-dialog-title" className="text-lg font-semibold text-base-content">{title}</h3>
          </div>
          <button
            onClick={onCancel}
            disabled={isPending}
            className="text-body-secondary transition-colors hover:text-base-content disabled:opacity-50"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-5">
          <p className="text-base-content/70">{description}</p>

           {confirmText && (
             <div className="mt-4">
               <label className="block text-sm font-medium text-base-content mb-2">
                 {UI_COPY.typeToConfirmDelete(confirmText)}
               </label>
               <input
                 ref={inputRef}
                 type="text"
                 value={inputValue}
                 onChange={(e) => setInputValue(e.target.value)}
                 placeholder={confirmText}
                 disabled={isPending}
                 className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
               />
             </div>
           )}

           {acknowledgement && (
             <label className="mt-4 flex items-start gap-3 cursor-pointer select-none">
               <input
                 type="checkbox"
                 checked={acknowledged}
                 onChange={(e) => setAcknowledged(e.target.checked)}
                 disabled={isPending}
                 className="checkbox checkbox-sm mt-0.5"
                 data-testid="confirmation-dialog-ack"
               />
               <span className="text-sm text-base-content/80">{acknowledgement}</span>
             </label>
           )}
        </div>

         {/* Footer */}
         <div className="flex justify-end gap-3 border-t ghost-divider modal-inner px-6 py-4">
           <button
             onClick={onCancel}
             disabled={isPending}
             className="action-btn-secondary"
           >
             {UI_COPY.cancel}
           </button>
           <button
             onClick={onConfirm}
             disabled={!canConfirm() || isPending}
             className={`px-4 py-2 rounded-xl ${styles.button} disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2`}
           >
             {isPending && (
               <div className="animate-spin rounded-full h-4 w-4 border-2 border-current border-t-transparent" />
             )}
             {isPending ? UI_COPY.processing : UI_COPY.confirm}
           </button>
         </div>
      </div>
    </div>
  );
}
