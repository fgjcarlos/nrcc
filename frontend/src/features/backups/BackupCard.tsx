import type { BackupSummary } from '../../api'
import { formatBytes } from '../../common/utils/format'

export function BackupCard({
  backup,
  confirming,
  isPending,
  onRestore,
  onCancel,
  onConfirm,
}: {
  backup: BackupSummary
  confirming: boolean
  isPending: boolean
  onRestore: () => void
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <article className="surface-panel border border-base-300/60 p-5" key={backup.id}>
      <div className="mb-4">
        <div className="flex flex-wrap items-center gap-2">
          <strong className="text-base text-base-content">{backup.id}</strong>
          <span className="rounded-full bg-base-300/60 px-2.5 py-1 text-xs text-base-content/70">
            {formatBytes(backup.archiveBytes)}
          </span>
        </div>
        <p className="text-sm text-base-content/75 mt-2">{backup.reason}</p>
        <p className="text-sm text-base-content/70 mt-1">{backup.archiveName}</p>
        <p className="text-xs text-base-content/60 mt-2">
          <span className="opacity-75">{backup.createdAt}</span>
        </p>
      </div>
      <div className="flex gap-2 justify-end">
        {confirming ? (
          <>
            <button
              className="action-btn-ghost"
              type="button"
              onClick={onCancel}
              disabled={isPending}
            >
              Cancel
            </button>
            <button
              className="action-btn-primary"
              type="button"
              onClick={onConfirm}
              disabled={isPending}
            >
              {isPending ? 'Restoring...' : 'Confirm restore'}
            </button>
          </>
        ) : (
          <button
            className="action-btn-secondary"
            type="button"
            onClick={onRestore}
            disabled={isPending}
          >
            Restore
          </button>
        )}
      </div>
    </article>
  )
}
