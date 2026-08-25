import { useState, useEffect, useRef, useMemo, useId } from 'react';
import { createPortal } from 'react-dom';
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
  const dialogRef = useRef<HTMLDivElement>(null);
  const cancelButtonRef = useRef<HTMLButtonElement>(null);
  const previouslyFocusedRef = useRef<HTMLElement | null>(null);
  const titleId = useId();

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

  // Reset gated state whenever the dialog opens.
  useEffect(() => {
    if (isOpen) {
      setInputValue('');
      setAcknowledged(false);
    }
  }, [isOpen]);

  // Move focus into the modal and return it to the invoking control on close.
  // The text gate is the safest useful target when present; otherwise Cancel
  // avoids placing initial focus on a destructive action.
  useEffect(() => {
    if (!isOpen) return;

    previouslyFocusedRef.current = document.activeElement as HTMLElement | null;
    const focusTimer = window.setTimeout(() => {
      (inputRef.current ?? cancelButtonRef.current)?.focus();
    }, 0);

    return () => {
      window.clearTimeout(focusTimer);
      const previouslyFocused = previouslyFocusedRef.current;
      previouslyFocusedRef.current = null;
      if (previouslyFocused?.isConnected) previouslyFocused.focus();
    };
  }, [isOpen]);

  // Keep keyboard focus inside the modal and preserve keyboard shortcuts.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!isOpen) return;

      if (e.key === 'Escape' && !isPending) {
        e.preventDefault();
        onCancel();
        return;
      }

      if (e.key === 'Tab') {
        const dialog = dialogRef.current;
        if (!dialog) return;

        const focusable = Array.from(
          dialog.querySelectorAll<HTMLElement>(
            'button:not([disabled]), input:not([disabled]), [href], [tabindex]:not([tabindex="-1"])'
          )
        );
        if (focusable.length === 0) {
          e.preventDefault();
          dialog.focus();
          return;
        }

        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        const active = document.activeElement;
        if (e.shiftKey && (active === first || !dialog.contains(active))) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && (active === last || !dialog.contains(active))) {
          e.preventDefault();
          first.focus();
        }
        return;
      }

      // Native buttons already synthesize one click for Enter. Handling that
      // key globally as well would submit twice.
      if (
        e.key === 'Enter' &&
        !e.repeat &&
        !isPending &&
        canConfirm() &&
        !(e.target instanceof HTMLButtonElement)
      ) {
        e.preventDefault();
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

  return createPortal(
    <div
      className="fixed inset-0 z-[100] flex items-center justify-center"
      data-confirmation-dialog-portal
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 modal-overlay"
        aria-hidden="true"
        onClick={isPending ? undefined : onCancel}
      />

      {/* Dialog */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={`relative surface-panel border ${styles.border} w-full max-w-md mx-4 overflow-hidden shadow-glow`}
      >
        {/* Header */}
        <div className="flex items-start justify-between border-b ghost-divider modal-inner p-6">
          <div className="flex items-center gap-3">
            <div className={`rounded-2xl bg-base-200/70 p-2 ${styles.icon}`}>
              <AlertTriangle className="w-5 h-5" />
            </div>
            <h3 id={titleId} className="text-lg font-semibold text-base-content">{title}</h3>
          </div>
          <button
            onClick={onCancel}
            disabled={isPending}
            aria-label="Close dialog"
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
             ref={cancelButtonRef}
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
    </div>,
    document.body
  );
}
