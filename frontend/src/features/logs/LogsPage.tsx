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
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Observability</p>
          <h2 className="page-title text-3xl mt-1">Logs</h2>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Stream runtime output in the same console style as the old dashboard.
          </p>
        </div>
      </header>

      {error ? (
        <InlineNotice
          tone="error"
          title="Logs unavailable"
          detail={formatErrorMessage(error, 'The runtime log stream could not be loaded.')}
        />
      ) : null}

      <article className="surface-card border border-base-300/60 p-6 md:p-7">
        <div className="flex items-center justify-between gap-4 mb-5">
          <div>
            <h3 className="section-title">Runtime logs</h3>
            <p className="mt-1 text-sm text-base-content/60">Plain-text output from the active Node-RED runtime.</p>
          </div>
          <span className="rounded-full bg-base-300/60 px-3 py-1 text-xs font-medium text-base-content/70">
            {logs.length} lines
          </span>
        </div>
        <div className="surface-panel border border-base-300/60 p-4 md:p-5">
          <div className="max-h-[32rem] space-y-2 overflow-auto font-mono text-sm leading-6">
            {loading ? <p className="text-sm text-base-content/60">Loading logs...</p> : null}
            {!loading && logs.length === 0 ? <p className="text-sm text-base-content/60">No logs captured yet.</p> : null}
            {logs.map((line, index) => (
              <div className="rounded-xl px-3 py-2 text-base-content/88 transition-colors hover:bg-base-300/40" key={`${index}-${line}`}>
                {line}
              </div>
            ))}
          </div>
        </div>
      </article>
    </>
  )
}
