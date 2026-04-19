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
    <article className="surface-card border section-card section-card--success p-6">
      <div>
        <div className="flex items-center gap-2 mb-3">
          <div className="h-2 w-2 rounded-full bg-success"></div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/55 font-semibold">Host</p>
        </div>
        <h3 className="text-xl font-semibold text-base-content">System info</h3>
        {systemLoading ? (
          <p className="text-base-content/60 text-sm mt-4">Loading system information...</p>
        ) : (
          <>
            <dl className="space-y-3 mt-4">
              <Detail label="Hostname" value={systemInfo?.hostname || 'N/A'} />
              <Detail label="OS" value={systemInfo ? `${systemInfo.goos}/${systemInfo.goarch}` : 'N/A'} />
              <Detail label="CPUs" value={systemInfo ? String(systemInfo.cpus) : 'N/A'} />
              <Detail label="Preferred access" value={systemInfo?.localAccess.url || 'N/A'} />
              <Detail label="Access mode" value={systemInfo ? `${systemInfo.localAccess.mode}${systemInfo.localAccess.configured ? ' (configured)' : ''}` : 'N/A'} />
              <Detail label="Updated" value={systemInfo?.timestamp || 'N/A'} />
            </dl>
            {systemInfo ? <p className="mt-4 text-sm text-base-content/65">{systemInfo.localAccess.message}</p> : null}
          </>
        )}
      </div>
    </article>
  )
}
