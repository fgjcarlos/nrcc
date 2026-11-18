import { useState, useEffect, useRef, useMemo, useId } from 'react';
import { useT } from '@/i18n';
import { Button } from './ui/Button';
import { Dialog, DialogBody, DialogFooter } from './ui/Dialog';

export type ConfirmationVariant = 'danger' | 'warning' | 'default';

interface ConfirmationDialogProps {
  isOpen: boolean;
  title: string;
  description: string;
  // Optional body slot rendered between the description and the
  // confirm/cancel footer. Slice 3 of #764 (AdvancedSettings) uses
  // this to render a redacted diff preview alongside the standard
  // description text. Callers that do not pass children get the
  // previous behaviour verbatim.
  children?: React.ReactNode;
  confirmText?: string;
  acknowledgement?: string;
  variant?: ConfirmationVariant;
  isPending?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

/** Destructive actions confirm on the danger button; everything else on the solid action. */
const CONFIRM_VARIANT = { danger: 'danger', warning: 'solid', default: 'solid' } as const;

export function ConfirmationDialog({
  isOpen,
  title,
  description,
  children,
  confirmText = '',
  acknowledgement,
  variant = 'default',
  isPending = false,
  onConfirm,
  onCancel,
}: ConfirmationDialogProps) {
  const [inputValue, setInputValue] = useState('');
  const [acknowledged, setAcknowledged] = useState(false);
  const { t } = useT();
  const inputRef = useRef<HTMLInputElement>(null);
  const cancelButtonRef = useRef<HTMLButtonElement>(null);
  const inputId = useId();

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

  // Enter confirms from anywhere in the dialog. Escape, the focus trap and
  // focus restore come from the shared Dialog. Native buttons already
  // synthesize one click for Enter; handling it here as well would submit twice.
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
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
  }, [isOpen, isPending, onConfirm, canConfirm]);

  return (
    <Dialog
      open={isOpen}
      onClose={onCancel}
      title={title}
      description={description}
      dismissible={!isPending}
      // The text gate is the safest useful target when present; otherwise
      // Cancel avoids placing initial focus on a destructive action.
      initialFocusRef={confirmText ? inputRef : cancelButtonRef}
    >
      {(confirmText || children || acknowledgement) && (
        <DialogBody className="space-y-4">
          {confirmText && (
            <div className="space-y-1">
              <label htmlFor={inputId} className="block text-xs text-ds-text-secondary">
                {t('typeToConfirmDelete', { identifier: confirmText })}
              </label>
              <input
                id={inputId}
                ref={inputRef}
                type="text"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                placeholder={confirmText}
                disabled={isPending}
                autoComplete="off"
                className="block w-full px-3 py-2 font-mono text-sm"
              />
            </div>
          )}

          {children}

          {acknowledgement && (
            <label className="flex cursor-pointer select-none items-start gap-3">
              <input
                type="checkbox"
                checked={acknowledged}
                onChange={(e) => setAcknowledged(e.target.checked)}
                disabled={isPending}
                className="checkbox checkbox-sm mt-0.5"
                data-testid="confirmation-dialog-ack"
              />
              <span className="text-sm text-ds-text-secondary">{acknowledgement}</span>
            </label>
          )}
        </DialogBody>
      )}

      <DialogFooter>
        <Button ref={cancelButtonRef} type="button" variant="secondary" onClick={onCancel} disabled={isPending}>
          {t('cancel')}
        </Button>
        <Button
          type="button"
          variant={CONFIRM_VARIANT[variant]}
          onClick={onConfirm}
          disabled={!canConfirm() || isPending}
        >
          {isPending && (
            <span
              className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent motion-reduce:animate-none"
              aria-hidden="true"
            />
          )}
          {isPending ? t('processing') : t('confirm')}
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
