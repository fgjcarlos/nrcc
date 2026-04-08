import type { DoctorReport } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage, formatCheckName } from '../../common/utils/format'
import { getStatusBadgeClass } from '../../common/utils/status'

export function DoctorTab({
  report,
  loading,
  error,
  onRefresh,
}: {
  report?: DoctorReport
  loading: boolean
  error: unknown
  onRefresh: () => Promise<void>
}) {
  return (
    <>
      {error ? (
        <InlineNotice
          tone="error"
          title="Doctor report unavailable"
          detail={formatErrorMessage(error, 'The doctor report could not be loaded.')}
        />
      ) : null}
      <div className="tab-content">
        <div className="diagnostics-header">
          <div className="diagnostics-status">
            {loading ? (
              <p className="muted">Loading doctor report...</p>
            ) : report ? (
              <>
                <div className={`status-badge ${getStatusBadgeClass(report.overall_status)}`}>
                  {report.overall_status.toUpperCase()}
                </div>
                <p className="muted">Generated at {new Date(report.generated_at).toLocaleString()}</p>
              </>
            ) : null}
          </div>
          <button
            className="ghost-button"
            type="button"
            onClick={onRefresh}
            disabled={loading}
          >
            Refresh
          </button>
        </div>

        {report && (
          <div className="checks-list">
            {report.checks.map((check) => (
              <div key={check.name} className={`check-item ${check.status}`}>
                <div className="check-status">
                  <span className="check-icon">
                    {check.status === 'pass' && '✅'}
                    {check.status === 'warn' && '⚠️'}
                    {check.status === 'fail' && '❌'}
                    {check.status === 'unknown' && '❓'}
                  </span>
                  <strong>{formatCheckName(check.name)}</strong>
                </div>
                <p>{check.message}</p>
                {check.details && (
                  <details className="check-details">
                    <summary>Details</summary>
                    <pre>{JSON.stringify(check.details, null, 2)}</pre>
                  </details>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  )
}
