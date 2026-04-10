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
  const getLevelBadgeClass = (level: string) => {
    switch (level) {
      case 'error':
        return 'badge-error'
      case 'warn':
        return 'badge-warning'
      case 'info':
        return 'badge-info'
      case 'debug':
      default:
        return 'badge-ghost'
    }
  }

  return (
    <>
      {error ? (
        <InlineNotice
          tone="error"
          title="Logs unavailable"
          detail={formatErrorMessage(error, 'The logs could not be loaded.')}
        />
      ) : null}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <p className="text-sm text-base-content opacity-60">{logs.length} logs</p>
          <button
            className="btn btn-ghost btn-sm"
            type="button"
            onClick={onRefresh}
            disabled={loading}
          >
            Refresh
          </button>
        </div>

        <div className="space-y-2">
          {loading ? (
            <p className="text-sm text-base-content opacity-60">Loading logs...</p>
          ) : logs.length === 0 ? (
            <p className="text-sm text-base-content opacity-60">No logs captured yet.</p>
          ) : (
            logs.map((log, idx) => (
              <div key={log.id || idx} className="flex items-start gap-3 p-3 bg-base-300 rounded border-l-2 border-base-200">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap mb-1">
                    <span className="text-xs text-base-content opacity-60">
                      {new Date(log.timestamp).toLocaleTimeString()}
                    </span>
                    <span className={`badge badge-sm ${getLevelBadgeClass(log.level)}`}>
                      {(log.level ?? '').toUpperCase()}
                    </span>
                    <span className="text-xs text-base-content opacity-60">{log.source}</span>
                  </div>
                  <div className="text-sm text-base-content break-words">{log.message}</div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </>
  )
}
