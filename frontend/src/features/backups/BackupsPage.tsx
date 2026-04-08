import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api, type BackupList } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import type { ToastTone } from '../../common/types'
import { BackupCard } from './BackupCard'

export function BackupsPage({
  backups,
  loading,
  error,
  onChanged,
}: {
  backups?: BackupList
  loading: boolean
  error: unknown
  onChanged: (message: string, tone: ToastTone) => Promise<void>
}) {
  const [restoreTarget, setRestoreTarget] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: api.createBackup,
    onSuccess: async () => {
      await onChanged('A manual backup was created successfully.', 'success')
    },
    onError: async (mutationError) => {
      await onChanged(formatErrorMessage(mutationError, 'The backup could not be created.'), 'error')
    },
  })

  const restoreMutation = useMutation({
    mutationFn: api.restoreBackup,
    onSuccess: async (result) => {
      setRestoreTarget(null)
      await onChanged(
        `Backup restored. Preventive backup created as ${result.preventiveBackupId}.`,
        'success',
      )
    },
    onError: async (mutationError) => {
      setRestoreTarget(null)
      await onChanged(formatErrorMessage(mutationError, 'The backup could not be restored.'), 'error')
    },
  })

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Backups</h2>
        </div>
        <div className="topbar-actions">
          <button
            className="primary-button"
            type="button"
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending}
          >
            {createMutation.isPending ? 'Creating...' : 'Create backup'}
          </button>
        </div>
      </header>

      {error ? (
        <InlineNotice
          tone="error"
          title="Backups unavailable"
          detail={formatErrorMessage(error, 'Backup history could not be loaded.')}
        />
      ) : null}

      <article className="panel">
        <div className="panel-header">
          <h3>Backup history</h3>
        </div>
        {loading ? <p className="muted">Loading backups...</p> : null}
        {!loading && (!backups || backups.items.length === 0) ? <p className="muted">No backups created yet.</p> : null}
        {backups?.items.length ? (
          <div className="backup-list">
            {backups.items.map((backup) => {
              const confirming = restoreTarget === backup.id
              return (
                <BackupCard
                  key={backup.id}
                  backup={backup}
                  confirming={confirming}
                  isPending={restoreMutation.isPending}
                  onRestore={() => setRestoreTarget(backup.id)}
                  onCancel={() => setRestoreTarget(null)}
                  onConfirm={() => restoreMutation.mutate(backup.id)}
                />
              )
            })}
          </div>
        ) : null}
      </article>
    </>
  )
}
