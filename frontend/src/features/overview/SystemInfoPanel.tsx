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
    <article className="panel">
      <div className="panel-header">
        <h3>System info</h3>
      </div>
      {systemLoading ? (
        <p className="muted">Loading system information...</p>
      ) : (
        <dl className="details-list">
          <Detail label="Hostname" value={systemInfo?.hostname || 'N/A'} />
          <Detail label="OS" value={systemInfo ? `${systemInfo.goos}/${systemInfo.goarch}` : 'N/A'} />
          <Detail label="CPUs" value={systemInfo ? String(systemInfo.cpus) : 'N/A'} />
          <Detail label="Updated" value={systemInfo?.timestamp || 'N/A'} />
        </dl>
      )}
    </article>
  )
}
