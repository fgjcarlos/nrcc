import { Detail } from '../../common/components'
import type { SystemInfo } from '../../api'

export function SystemInfoPanel({
  systemInfo,
  systemLoading,
}: {
  systemInfo?: SystemInfo
  systemLoading: boolean
}) {
  return (
    <article className="surface-card border border-base-300 p-6">
      <div>
        <p className="text-xs uppercase tracking-[0.24em] text-base-content/55">Host</p>
        <h3 className="mt-2 text-xl font-semibold text-base-content">System info</h3>
        {systemLoading ? (
          <p className="text-base-content/60 text-sm">Loading system information...</p>
        ) : (
          <dl className="space-y-3">
            <Detail label="Hostname" value={systemInfo?.hostname || 'N/A'} />
            <Detail label="OS" value={systemInfo ? `${systemInfo.goos}/${systemInfo.goarch}` : 'N/A'} />
            <Detail label="CPUs" value={systemInfo ? String(systemInfo.cpus) : 'N/A'} />
            <Detail label="Updated" value={systemInfo?.timestamp || 'N/A'} />
          </dl>
        )}
      </div>
    </article>
  )
}
