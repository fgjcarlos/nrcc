import type { LogEntry } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'

export function LogsTab({
  logs,
  loading,
  error,
  onRefresh,
}: {
  logs: LogEntry[]
  loading: boolean
  error: unknown
  onRefresh: () => Promise<void>
}) {
  return (
    <>
      {error ? (
        <InlineNotice
          tone="error"
          title="Logs unavailable"
          detail={formatErrorMessage(error, 'The logs could not be loaded.')}
        />
      ) : null}
      <div className="tab-content">
        <div className="diagnostics-header">
          <p className="muted">{logs.length} logs</p>
          <button
            className="ghost-button"
            type="button"
            onClick={onRefresh}
            disabled={loading}
          >
            Refresh
          </button>
        </div>

        <div className="logs-list">
          {loading ? (
            <p className="muted">Loading logs...</p>
          ) : logs.length === 0 ? (
            <p className="muted">No logs captured yet.</p>
          ) : (
            logs.map((log, idx) => (
              <div key={log.id || idx} className={`log-item level-${log.level}`}>
                <div className="log-meta">
                  <span className="log-timestamp">
                    {new Date(log.timestamp).toLocaleTimeString()}
                  </span>
                  <span className={`log-badge level-${log.level}`}>
                    {log.level.toUpperCase()}
                  </span>
                  <span className="log-source">{log.source}</span>
                </div>
                <div className="log-message">{log.message}</div>
              </div>
            ))
          )}
        </div>
      </div>
    </>
  )
}
