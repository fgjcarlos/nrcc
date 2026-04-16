import type { DoctorReport } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage, formatCheckName } from '../../common/utils/format'
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
  const getStatusBadgeDaisyUI = (status: string) => {
    switch (status) {
      case 'pass':
        return 'badge-success'
      case 'fail':
        return 'badge-error'
      case 'warn':
        return 'badge-warning'
      case 'unknown':
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
              <p className="text-sm text-base-content opacity-60">Loading doctor report...</p>
            ) : report ? (
              <>
                <div className={`badge ${getStatusBadgeDaisyUI(report.overall_status)} badge-lg`}>
                  {(report.overall_status ?? '').toUpperCase()}
                </div>
                <p className="text-sm text-base-content opacity-60">Generated at {new Date(report.generated_at).toLocaleString()}</p>
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
              <div key={check.name} className="list-shell p-4 md:p-5">
                <div className="flex items-start gap-3 mb-2">
                  <span className="text-lg flex-shrink-0">
                    {check.status === 'pass' && '✅'}
                    {check.status === 'warn' && '⚠️'}
                    {check.status === 'fail' && '❌'}
                    {check.status === 'unknown' && '❓'}
                  </span>
                  <div className="flex-1">
                    <strong className="text-base-content block">{formatCheckName(check.name)}</strong>
                    <span className={`badge badge-sm ${getStatusBadgeDaisyUI(check.status)} mt-1`}>
                      {check.status.toUpperCase()}
                    </span>
                  </div>
                </div>
                <p className="text-sm text-base-content opacity-85 mb-2">{check.message}</p>
                {check.details && (
                  <details className="text-xs">
                    <summary className="cursor-pointer font-semibold opacity-75 hover:opacity-100">Details</summary>
                    <pre className="code-block-bg mt-2 overflow-auto max-h-40 text-xs opacity-75">{JSON.stringify(check.details, null, 2)}</pre>
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
