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
    <article className="surface-card border section-card section-card--info p-6">
      <div>
        <div className="flex items-center gap-2 mb-3">
          <div className="h-2 w-2 rounded-full bg-info"></div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/55 font-semibold">Runtime</p>
        </div>
        <h3 className="text-xl font-semibold text-base-content">Runtime details</h3>
        <dl className="space-y-3 mt-4">
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
