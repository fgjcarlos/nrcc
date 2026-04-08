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
    <article className="backup-card" key={backup.id}>
      <div className="backup-card-copy">
        <strong>{backup.id}</strong>
        <p>{backup.reason}</p>
        <p>{backup.archiveName}</p>
        <p>
          {formatBytes(backup.archiveBytes)} • {backup.createdAt}
        </p>
      </div>
      <div className="backup-card-actions">
        {confirming ? (
          <>
            <button
              className="ghost-button"
              type="button"
              onClick={onCancel}
              disabled={isPending}
            >
              Cancel
            </button>
            <button
              className="primary-button"
              type="button"
              onClick={onConfirm}
              disabled={isPending}
            >
              {isPending ? 'Restoring...' : 'Confirm restore'}
            </button>
          </>
        ) : (
          <button
            className="ghost-button"
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
