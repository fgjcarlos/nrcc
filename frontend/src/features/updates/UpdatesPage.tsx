import type { UpdateStatus, OperationStatus } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import type { ToastTone } from '../../common/types'
import { UpdateCard } from './UpdateCard'

export function UpdatesPage({
  updateStatus,
  loading,
  error,
  operationStatus,
  onChanged,
}: {
  updateStatus?: UpdateStatus
  loading: boolean
  error: unknown
  operationStatus?: OperationStatus
  onChanged: (message: string, tone: ToastTone) => Promise<void>
}) {
  return (
    <>
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-8">
        <div>
          <p className="text-sm font-semibold text-base-content/70 uppercase tracking-wide">Runtime</p>
          <h2 className="text-3xl font-bold text-base-content mt-1">Updates</h2>
        </div>
      </header>

      {error ? (
        <InlineNotice
          tone="error"
          title="Update status unavailable"
          detail={formatErrorMessage(error, 'Node-RED update information could not be loaded.')}
        />
      ) : null}

      {operationStatus?.busy ? (
        <InlineNotice
          tone="warn"
          title="System busy"
          detail={
            (operationStatus.type ? `${operationStatus.type} in progress` : 'Another operation is in progress') +
            (operationStatus.detail ? `: ${operationStatus.detail}` : '.')
          }
        />
      ) : null}

      <article className="card bg-base-200 shadow">
        <div className="card-body">
          <h3 className="card-title text-2xl">Node-RED update</h3>
          {loading ? <p className="text-sm text-base-content/60">Loading update status...</p> : null}
          {updateStatus ? (
            <UpdateCard
              updateStatus={updateStatus}
              operationStatus={operationStatus}
              onChanged={onChanged}
            />
          ) : null}
        </div>
      </article>
    </>
  )
}
