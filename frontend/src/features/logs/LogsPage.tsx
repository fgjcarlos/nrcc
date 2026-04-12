import type { LogEntry } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'

export function LogsPage({
  logs,
  loading,
  error,
}: {
  logs: string[]
  loading: boolean
  error: unknown
}) {
  return (
    <>
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-8">
        <div>
          <p className="text-sm font-semibold text-base-content/70 uppercase tracking-wide">Runtime</p>
          <h2 className="text-3xl font-bold text-base-content mt-1">Logs</h2>
        </div>
      </header>

      {error ? (
        <InlineNotice
          tone="error"
          title="Logs unavailable"
          detail={formatErrorMessage(error, 'The runtime log stream could not be loaded.')}
        />
      ) : null}

      <article className="card bg-base-200 shadow">
        <div className="card-body">
          <h3 className="card-title text-2xl">Runtime logs</h3>
          <div className="log-output">
            {loading ? <p className="text-sm text-base-content/60">Loading logs...</p> : null}
            {!loading && logs.length === 0 ? <p className="text-sm text-base-content/60">No logs captured yet.</p> : null}
            {logs.map((line, index) => (
              <div className="log-line" key={`${index}-${line}`}>
                {line}
              </div>
            ))}
          </div>
        </div>
      </article>
    </>
  )
}
