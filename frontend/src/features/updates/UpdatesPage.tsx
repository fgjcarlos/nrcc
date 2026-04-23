import { InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import { useUpdatesData } from './useUpdatesData'
import { UpdateCard } from './UpdateCard'

export function UpdatesPage() {
  const { updateStatus, loading, error, operationStatus } = useUpdatesData()

  return (
    <>
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-8">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Maintenance</p>
          <h2 className="page-title text-3xl mt-1">Updates</h2>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Check the installed runtime against the available release and apply updates with rollback protection.
          </p>
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

      <article className="surface-card border border-base-300/60 p-6 md:p-7">
        <div className="mb-5">
          <h3 className="section-title">Node-RED update</h3>
          <p className="mt-1 text-sm text-base-content/60">Version status and update execution flow.</p>
        </div>
          {loading ? <p className="text-sm text-base-content/60">Loading update status...</p> : null}
          {updateStatus ? (
            <UpdateCard
              updateStatus={updateStatus}
              operationStatus={operationStatus}
            />
          ) : null}
      </article>
    </>
  )
}
