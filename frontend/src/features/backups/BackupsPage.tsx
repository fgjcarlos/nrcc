import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api'
import { InlineNotice, LoadingState, EmptyState } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import type { ToastTone } from '../../common/types'
import { useToasts } from '../shell/useToasts'
import { useBackupsData } from './useBackupsData'
import { BackupCard } from './BackupCard'

export function BackupsPage() {
  const { backups, loading, error, operationStatus } = useBackupsData()
  const queryClient = useQueryClient()
  const { pushToast } = useToasts()
  const [restoreTarget, setRestoreTarget] = useState<string | null>(null)
  const busy = operationStatus?.busy ?? false

  const createMutation = useMutation({
    mutationFn: api.createBackup,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['backups'] })
      await queryClient.invalidateQueries({ queryKey: ['operations-status'] })
      pushToast({
        tone: 'success',
        title: 'Backups updated',
        detail: 'A manual backup was created successfully.',
      })
    },
    onError: async (mutationError) => {
      await queryClient.invalidateQueries({ queryKey: ['backups'] })
      await queryClient.invalidateQueries({ queryKey: ['operations-status'] })
      pushToast({
        tone: 'error',
        title: 'Backup action failed',
        detail: formatErrorMessage(mutationError, 'The backup could not be created.'),
      })
    },
  })

  const restoreMutation = useMutation({
    mutationFn: api.restoreBackup,
    onSuccess: async (result) => {
      setRestoreTarget(null)
      await queryClient.invalidateQueries({ queryKey: ['backups'] })
      await queryClient.invalidateQueries({ queryKey: ['operations-status'] })
      pushToast({
        tone: 'success',
        title: 'Backups updated',
        detail: `Backup restored. Preventive backup created as ${result.preventiveBackupId}.`,
      })
    },
    onError: async (mutationError) => {
      setRestoreTarget(null)
      await queryClient.invalidateQueries({ queryKey: ['backups'] })
      await queryClient.invalidateQueries({ queryKey: ['operations-status'] })
      pushToast({
        tone: 'error',
        title: 'Backup action failed',
        detail: formatErrorMessage(mutationError, 'The backup could not be restored.'),
      })
    },
  })

  return (
    <>
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-8">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Recovery</p>
          <h2 className="page-title text-3xl mt-1">Backups</h2>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Local snapshots of runtime state, presented in the same recovery-oriented layout as the old app.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            className="action-btn-primary"
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

      {busy ? (
        <InlineNotice
          tone="warn"
          title="System busy"
          detail={
            (operationStatus?.type ? `${operationStatus.type} in progress` : 'Another operation is in progress') +
            (operationStatus?.detail ? `: ${operationStatus.detail}` : '.')
          }
        />
      ) : null}

      <article className="surface-card border border-base-300/60 p-6 md:p-7">
        <div className="mb-5">
          <h3 className="section-title">Backup history</h3>
          <p className="mt-1 text-sm text-base-content/60">Restore points created manually and automatically by the backend.</p>
        </div>
          {loading ? <LoadingState message="Loading backups..." /> : null}
          {!loading && (!backups || backups.items.length === 0) ? (
            <EmptyState
              title="No backups created yet"
              description="Create a manual backup or wait for an automatic one to appear here."
              action={{
                label: 'Create backup',
                onClick: () => createMutation.mutate(),
                disabled: createMutation.isPending,
              }}
            />
          ) : null}
          {backups?.items.length ? (
            <div className="space-y-4">
              {backups.items.map((backup) => {
                const confirming = restoreTarget === backup.id
                return (
                  <BackupCard
                    key={backup.id}
                    backup={backup}
                    confirming={confirming}
                    isPending={busy || restoreMutation.isPending}
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
