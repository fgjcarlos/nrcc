import { useId, useRef, useState, type FormEvent } from 'react';
import { Button, Dialog, DialogBody, DialogFooter, Input } from '@/shared/components/ui';
import { useT } from '@/i18n';
import type { BackupSummary } from '@/features/backups/services';
import { formatBackupDate, formatBackupSize, getBackupDisplayName } from '@/features/backups/lib/formatters';
import { BackupTypeChip } from './BackupTypeChip';

export type BackupAction = 'restore' | 'delete';

// The order RestoreWithSafetyBackup runs (internal/service/backup.go): stop, snapshot, verify + swap, start.
const RESTORE_STEPS = ['stop', 'snapshot', 'replace', 'start'] as const;

interface BackupActionDialogProps {
  action: BackupAction;
  backup: BackupSummary;
  isPending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * Restore / delete confirmation. Same guard as the shared ConfirmationDialog
 * (confirm stays disabled until the typed token matches exactly, Enter only
 * confirms once it does, the dialog cannot be dismissed while pending), but
 * with a short localized word as the token, the backup's identity, the real
 * restore sequence and a confirm button that names the backup.
 */
export function BackupActionDialog({ action, backup, isPending, onConfirm, onCancel }: BackupActionDialogProps) {
  const { t, i18n } = useT();
  const [typed, setTyped] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const inputId = useId();
  const stepsId = useId();
  const k = (key: string) => `backups:dialog.${action}.${key}`;
  const token = t(k('token'));
  const date = formatBackupDate(backup.createdAt, i18n.language) ?? t('backups:dateUnavailable');
  const canConfirm = typed === token && !isPending;

  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (canConfirm) onConfirm();
  };

  const identity = [
    { label: t('backups:dialog.name'), value: getBackupDisplayName(backup) },
    { label: t('backups:dialog.created'), value: date },
    { label: t('backups:dialog.type'), value: <BackupTypeChip type={backup.type} /> },
    { label: t('backups:dialog.size'), value: formatBackupSize(backup.totalSize) },
    { label: t('backups:dialog.files'), value: String(backup.fileCount) },
  ];

  return (
    <Dialog
      open
      onClose={onCancel}
      title={t(k('title'))}
      description={t(k('description'))}
      dismissible={!isPending}
      closeOnBackdrop={false}
      initialFocusRef={inputRef}
    >
      <form onSubmit={submit} className="flex min-h-0 flex-1 flex-col" data-backup-action={action}>
        <DialogBody className="space-y-4">
          <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 border-y border-ds-border-subtle py-2">
            {identity.map((row) => (
              <div key={row.label} className="contents">
                <dt className="text-xs leading-6 text-ds-text-secondary">{row.label}</dt>
                <dd className="min-w-0 break-words text-sm leading-6 tabular-nums text-ds-text-primary">{row.value}</dd>
              </div>
            ))}
          </dl>

          {action === 'restore' && (
            <section aria-labelledby={stepsId} className="space-y-2">
              <h3 id={stepsId} className="text-sm font-semibold text-ds-text-primary">
                {t('backups:dialog.restore.stepsTitle')}
              </h3>
              <ol className="space-y-2">
                {RESTORE_STEPS.map((step, index) => (
                  <li key={step} className="grid grid-cols-[1.5rem_minmax(0,1fr)] gap-2 text-sm text-ds-text-secondary">
                    <span aria-hidden="true" className="font-mono text-xs font-semibold leading-5 text-ds-text-primary">
                      {index + 1}
                    </span>
                    <span>{t(`backups:dialog.restore.steps.${step}`)}</span>
                  </li>
                ))}
              </ol>
            </section>
          )}

          <Input
            ref={inputRef}
            id={inputId}
            label={t('backups:dialog.tokenLabel', { token })}
            value={typed}
            onChange={(event) => setTyped(event.target.value)}
            placeholder={token}
            disabled={isPending}
            autoComplete="off"
            spellCheck={false}
            className="font-mono"
          />
        </DialogBody>

        <DialogFooter>
          <Button type="button" variant="secondary" onClick={onCancel} disabled={isPending}>
            {t('common:cancel')}
          </Button>
          <Button type="submit" variant={action === 'delete' ? 'danger' : 'solid'} disabled={!canConfirm}>
            {isPending ? t('common:processing') : t(k('confirm'), { date })}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  );
}
