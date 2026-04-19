import type { DoctorReport } from '../../api'
import { InlineNotice, LoadingState } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
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
  const getCheckStatusBadge = (status: string) => {
    switch (status) {
      case 'pass':
        return 'badge-success'
      case 'warn':
        return 'badge-warning'
      case 'fail':
        return 'badge-error'
      default:
        return 'badge-ghost'
    }
  }

  const getOverallStatusBadge = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'badge-success'
      case 'degraded':
        return 'badge-warning'
      case 'critical':
        return 'badge-error'
      default:
        return 'badge-ghost'
    }
  }

  return (
    <>
      {error ? (
        <InlineNotice
          tone="error"
          title="Doctor report unavailable"
          detail={formatErrorMessage(error, 'The doctor report could not be loaded.')}
        />
      ) : null}
      <div className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-center gap-3 sm:gap-4">
            {loading ? (
              <LoadingState message="Loading doctor report..." size="sm" />
            ) : report ? (
              <>
                <div className={`badge ${getOverallStatusBadge(report.overallStatus)} badge-lg`}>
                  {(report.overallStatus ?? '').toUpperCase()}
                </div>
                <p className="text-sm text-base-content opacity-60">Generated at {new Date(report.generatedAt).toLocaleString()}</p>
              </>
            ) : null}
          </div>
          <button
            className="action-btn-ghost self-start sm:self-auto"
            type="button"
            onClick={onRefresh}
            disabled={loading}
          >
            Refresh
          </button>
        </div>

        {report && (
          <div className="space-y-3">
            {report.checks.map((check) => (
              <div key={check.id} className="list-shell p-4 md:p-5">
                <div className="flex items-start gap-3 mb-2">
                  <span className="text-lg flex-shrink-0">
                    {check.status === 'pass' && '✅'}
                    {check.status === 'warn' && '⚠️'}
                    {check.status === 'fail' && '❌'}
                  </span>
                  <div className="flex-1">
                    <strong className="text-base-content block">{check.label}</strong>
                    <span className={`badge badge-sm ${getCheckStatusBadge(check.status)} mt-1`}>
                      {check.status.toUpperCase()}
                    </span>
                  </div>
                </div>
                <p className="text-sm text-base-content opacity-85 mb-2">{check.message}</p>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  )
}
