import { useEffect, useId, useRef, type HTMLAttributes, type ReactNode, type RefObject } from 'react';
import { createPortal } from 'react-dom';
import { X } from 'lucide-react';
import { cn } from '@/shared/lib/utils';
import { useT } from '@/i18n';
import { useModalKeyboard } from '@/shared/hooks/useModalKeyboard';
import { Button } from './Button';

const SIZES = { sm: 'max-w-sm', md: 'max-w-md', lg: 'max-w-3xl' } as const;

export interface DialogProps {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  /** Rendered under the title and wired to aria-describedby. */
  description?: ReactNode;
  children?: ReactNode;
  size?: keyof typeof SIZES;
  /** False while an action is in flight: Escape, the backdrop and the close button stop dismissing. */
  dismissible?: boolean;
  /** False when a stray backdrop click would throw away typed work. */
  closeOnBackdrop?: boolean;
  /** Initial focus target; defaults to the first control in the content, never the close button. */
  initialFocusRef?: RefObject<HTMLElement | null>;
}

/**
 * The shared modal dialog (KEYBOARD_NAV.md): portaled to document.body above
 * page drawers, focus trapped, Escape to close, focus restored to the opener,
 * document scroll locked. Compose the content with DialogBody and DialogFooter
 * (a form can wrap both so its submit button lives in the footer).
 */
export function Dialog({
  open,
  onClose,
  title,
  description,
  children,
  size = 'md',
  dismissible = true,
  closeOnBackdrop = true,
  initialFocusRef,
}: DialogProps) {
  const { t } = useT();
  const titleId = useId();
  const descriptionId = useId();
  const dialogRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  useModalKeyboard({
    isOpen: open,
    containerRef: dialogRef,
    onEscape: onClose,
    initialFocusRef,
    initialFocusScopeRef: contentRef,
    escapeDisabled: !dismissible,
  });

  // The dialog is portaled outside the page shell, so lock the real document scroll.
  useEffect(() => {
    if (!open) return;
    const { body, documentElement } = document;
    const previous = [body.style.overflow, documentElement.style.overflow];
    body.style.overflow = 'hidden';
    documentElement.style.overflow = 'hidden';
    return () => {
      [body.style.overflow, documentElement.style.overflow] = previous;
    };
  }, [open]);

  if (!open) return null;

  return createPortal(
    // data-confirmation-dialog-portal: the top-layer hook pinned by #721 tests.
    <div
      className="modal-overlay motion-scrim z-[100]"
      data-confirmation-dialog-portal
      onClick={(event) => {
        if (event.target === event.currentTarget && dismissible && closeOnBackdrop) onClose();
      }}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        tabIndex={-1}
        className={cn(
          'motion-overlay flex max-h-full w-full flex-col overflow-hidden rounded-box border border-ds-border-default bg-ds-bg-elevated text-ds-text-primary shadow-glow',
          SIZES[size]
        )}
      >
        <div className="flex items-start justify-between gap-4 px-6 pt-5 pb-4">
          <div className="min-w-0">
            <h2 id={titleId} className="text-xl font-bold leading-7">
              {title}
            </h2>
            {description && (
              <p id={descriptionId} className="mt-1 text-sm text-ds-text-secondary">
                {description}
              </p>
            )}
          </div>
          <Button
            type="button"
            variant="ghost"
            onClick={onClose}
            disabled={!dismissible}
            aria-label={t('common:closeDialog')}
            className="-mr-2 min-w-11 shrink-0 px-2 md:min-w-0"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </Button>
        </div>
        <div ref={contentRef} className="flex min-h-0 flex-1 flex-col">
          {children}
        </div>
      </div>
    </div>,
    document.body
  );
}

/** Scrollable dialog content, aligned with the title. */
export function DialogBody({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('min-h-0 flex-1 overflow-y-auto px-6 pb-5 text-sm', className)} {...props} />;
}

/** Dialog actions on a hairline rule; the affirmative action goes last. */
export function DialogFooter({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('flex flex-wrap justify-end gap-2 border-t border-ds-border-default px-6 py-4', className)}
      {...props}
    />
  );
}
