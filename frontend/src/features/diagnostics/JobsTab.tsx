import type { JobRecord } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage, formatDuration } from '../../common/utils/format'

export function JobsTab({
  jobs,
  loading,
  error,
  onRefresh,
}: {
  jobs: JobRecord[]
  loading: boolean
  error: unknown
  onRefresh: () => Promise<void>
}) {
  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case 'completed':
        return 'badge-success'
      case 'failed':
        return 'badge-error'
      case 'running':
        return 'badge-info'
      case 'pending':
      default:
        return 'badge-ghost'
    }
  }

  return (
    <>
      {error ? (
        <InlineNotice
          tone="error"
          title="Jobs unavailable"
          detail={formatErrorMessage(error, 'The jobs history could not be loaded.')}
        />
      ) : null}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <p className="text-sm text-base-content opacity-60">{jobs.length} jobs</p>
          <button
            className="btn btn-ghost btn-sm"
            type="button"
            onClick={onRefresh}
            disabled={loading}
          >
            Refresh
          </button>
        </div>

        <div className="overflow-x-auto">
          {loading ? (
            <p className="text-sm text-base-content opacity-60">Loading jobs...</p>
          ) : jobs.length === 0 ? (
            <p className="text-sm text-base-content opacity-60">No jobs recorded yet.</p>
          ) : (
            <table className="table table-compact w-full">
              <thead>
                <tr className="bg-base-300">
                  <th className="text-base-content">Type</th>
                  <th className="text-base-content">Status</th>
                  <th className="text-base-content">Started</th>
                  <th className="text-base-content">Duration</th>
                  <th className="text-base-content">Summary</th>
                </tr>
              </thead>
              <tbody>
                {jobs.map((job) => (
                  <tr key={job.id} className="hover:bg-base-300">
                    <td className="text-sm text-base-content font-semibold">{job.type}</td>
                    <td>
                      <span className={`badge badge-sm ${getStatusBadgeClass(job.status)}`}>
                        {(job.status ?? '').toUpperCase()}
                      </span>
                    </td>
                    <td className="text-xs text-base-content opacity-75">{new Date(job.started_at).toLocaleString()}</td>
                    <td className="text-xs text-base-content opacity-75">
                      {job.finished_at
                        ? formatDuration(
                            new Date(job.finished_at).getTime() -
                              new Date(job.started_at).getTime(),
                          )
                        : '—'}
                    </td>
                    <td className="text-xs text-base-content opacity-85 max-w-xs truncate">
                      {job.summary || job.error || '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
    </>
  )
}
