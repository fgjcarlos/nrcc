import { useState, useCallback } from 'react';

/**
 * Hook to manage confirmation dialog state for a single pending item
 * Encapsulates state management of: pendingItem, isOpen, open/close handlers
 * Usage: const { open, close, isOpen, pendingItem } = useConfirmationDialog<User>();
 */
export function useConfirmationDialog<T>() {
  const [pendingItem, setPendingItem] = useState<T | null>(null);
  const [isOpen, setIsOpen] = useState(false);

  const open = useCallback((item: T) => {
    setPendingItem(item);
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    setIsOpen(false);
    setPendingItem(null);
  }, []);

  return {
    pendingItem,
    isOpen,
    open,
    close,
  };
}
