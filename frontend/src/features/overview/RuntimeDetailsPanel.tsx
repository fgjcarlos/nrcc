import { Detail } from '../../common/components'
import { formatUptime } from '../../common/utils/format'
import type { RuntimeStatus } from '../../api'

export function RuntimeDetailsPanel({
  runtime,
  runtimeLoading,
}: {
  runtime?: RuntimeStatus
  runtimeLoading: boolean
}) {
  return (
    <article className="card bg-base-200 shadow-elevation-2 rounded-lg">
      <div className="card-body">
        <h3 className="card-title text-lg font-semibold text-base-content">Runtime details</h3>
        <dl className="space-y-3">
          <Detail label="PID" value={runtime?.pid ? String(runtime.pid) : 'N/A'} />
          <Detail label="Port" value={runtime?.port ? String(runtime.port) : 'N/A'} />
          <Detail label="Started at" value={runtime?.startedAt || 'N/A'} />
          <Detail label="Uptime" value={runtimeLoading ? 'Loading...' : formatUptime(runtime?.uptimeSec ?? 0)} />
          <Detail label="Data dir" value={runtime?.dataDir || 'N/A'} />
          <Detail label="Last error" value={runtime?.lastError || 'None'} />
        </dl>
      </div>
    </article>
  )
}
