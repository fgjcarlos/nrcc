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
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Logs</h2>
        </div>
      </header>

      {error ? (
        <InlineNotice
          tone="error"
          title="Logs unavailable"
          detail={formatErrorMessage(error, 'The runtime log stream could not be loaded.')}
        />
      ) : null}

      <article className="panel logs-panel">
        <div className="panel-header">
          <h3>Runtime logs</h3>
        </div>
        <div className="log-output">
          {loading ? <p className="muted">Loading logs...</p> : null}
          {!loading && logs.length === 0 ? <p className="muted">No logs captured yet.</p> : null}
          {logs.map((line, index) => (
            <div className="log-line" key={`${index}-${line}`}>
              {line}
            </div>
          ))}
        </div>
      </article>
    </>
  )
}
