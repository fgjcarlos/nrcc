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
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-8">
        <div>
          <p className="text-sm font-semibold text-base-content/70 uppercase tracking-wide">Runtime</p>
          <h2 className="text-3xl font-bold text-base-content mt-1">Backups</h2>
        </div>
        <div className="flex gap-2">
          <button
            className="btn btn-primary"
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

      <article className="card bg-base-200 shadow">
        <div className="card-body">
          <h3 className="card-title text-2xl">Backup history</h3>
          {loading ? <p className="text-sm text-base-content/60">Loading backups...</p> : null}
          {!loading && (!backups || backups.items.length === 0) ? <p className="text-sm text-base-content/60">No backups created yet.</p> : null}
          {backups?.items.length ? (
            <div className="space-y-4">
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
        </div>
      </article>
    </>
  )
}
