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
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Updates</h2>
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

      <article className="panel">
        <div className="panel-header">
          <h3>Node-RED update</h3>
        </div>
        {loading ? <p className="muted">Loading update status...</p> : null}
        {updateStatus ? (
          <UpdateCard
            updateStatus={updateStatus}
            operationStatus={operationStatus}
            onChanged={onChanged}
          />
        ) : null}
      </article>
    </>
  )
}
