import { useEffect, useRef, type RefObject } from 'react';

/**
 * useModalKeyboard — issue #889
 *
 * Keyboard contract for modal dialogs, following the pattern already used
 * by ConfirmationDialog:
 *  - on open, remember the focused element and move focus into the dialog
 *    (the `initialFocusRef` target, else the first focusable control, else
 *    the dialog container itself);
 *  - Escape calls `onEscape` unless `escapeDisabled` is set;
 *  - Tab and Shift+Tab wrap inside the dialog;
 *  - on close or unmount, focus returns to the remembered element.
 */

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(', ');

export interface UseModalKeyboardOptions {
  isOpen: boolean;
  containerRef: RefObject<HTMLElement | null>;
  onEscape: () => void;
  initialFocusRef?: RefObject<HTMLElement | null>;
  escapeDisabled?: boolean;
}

function focusableWithin(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR));
}

export function useModalKeyboard({
  isOpen,
  containerRef,
  onEscape,
  initialFocusRef,
  escapeDisabled = false,
}: UseModalKeyboardOptions): void {
  // Keep the latest callback without re-running the focus effect.
  const onEscapeRef = useRef(onEscape);
  useEffect(() => {
    onEscapeRef.current = onEscape;
  }, [onEscape]);

  // Initial focus and focus restore.
  useEffect(() => {
    if (!isOpen) return;

    const previouslyFocused = document.activeElement as HTMLElement | null;
    const container = containerRef.current;
    if (container) {
      const target =
        initialFocusRef?.current ?? focusableWithin(container)[0] ?? container;
      target.focus();
    }

    return () => {
      if (previouslyFocused?.isConnected) previouslyFocused.focus();
    };
    // Refs are stable, so focus moves only when the dialog opens or closes.
  }, [isOpen, containerRef, initialFocusRef]);

  // Escape and focus trap.
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (escapeDisabled) return;
        e.preventDefault();
        onEscapeRef.current();
        return;
      }

      if (e.key !== 'Tab') return;
      const container = containerRef.current;
      if (!container) return;

      const focusable = focusableWithin(container);
      if (focusable.length === 0) {
        e.preventDefault();
        container.focus();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const active = document.activeElement;
      if (e.shiftKey && (active === first || !container.contains(active))) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && (active === last || !container.contains(active))) {
        e.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, escapeDisabled, containerRef]);
}
