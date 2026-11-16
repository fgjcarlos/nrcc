import { useRef } from 'react';
import { RefreshCw } from 'lucide-react';
import { useT } from '@/i18n';
import { Button, Dialog, DialogFooter } from '@/shared/components/ui';

interface RestartConfirmationModalProps {
  isOpen: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export function RestartConfirmationModal({ isOpen, onCancel, onConfirm }: RestartConfirmationModalProps) {
  const { t } = useT();
  const cancelButtonRef = useRef<HTMLButtonElement>(null);

  // The shared Dialog portals to document.body (#704), so the backdrop
  // covers the viewport rather than the scrolling main container. Initial
  // focus lands on the non-destructive Cancel action.
  return (
    <Dialog
      open={isOpen}
      onClose={onCancel}
      title={t('dashboard:restart.title')}
      description={t('dashboard:restart.description')}
      size="sm"
      initialFocusRef={cancelButtonRef}
    >
      <DialogFooter>
        <Button ref={cancelButtonRef} type="button" variant="secondary" onClick={onCancel}>
          {t('common:cancel')}
        </Button>
        <Button type="button" onClick={onConfirm}>
          <RefreshCw className="h-4 w-4" aria-hidden="true" />
          {t('dashboard:restart.confirm')}
        </Button>
      </DialogFooter>
    </Dialog>
  );
}
