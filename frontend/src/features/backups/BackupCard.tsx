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
    <article className="card bg-base-200 p-6" key={backup.id}>
      <div className="mb-4">
        <strong className="text-base text-base-content">{backup.id}</strong>
        <p className="text-sm text-base-content opacity-75 mt-2">{backup.reason}</p>
        <p className="text-sm text-base-content opacity-75 mt-1">{backup.archiveName}</p>
        <p className="text-xs text-base-content opacity-60 mt-2">
          <span className="badge badge-ghost">{formatBytes(backup.archiveBytes)}</span>
          <span className="ml-2 opacity-75">{backup.createdAt}</span>
        </p>
      </div>
      <div className="flex gap-2 justify-end">
        {confirming ? (
          <>
            <button
              className="btn btn-ghost btn-sm"
              type="button"
              onClick={onCancel}
              disabled={isPending}
            >
              Cancel
            </button>
            <button
              className="btn btn-primary btn-sm"
              type="button"
              onClick={onConfirm}
              disabled={isPending}
            >
              {isPending ? 'Restoring...' : 'Confirm restore'}
            </button>
          </>
        ) : (
          <button
            className="btn btn-ghost btn-sm"
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
