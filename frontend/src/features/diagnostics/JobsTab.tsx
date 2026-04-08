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
  return (
    <>
      {error ? (
        <InlineNotice
          tone="error"
          title="Jobs unavailable"
          detail={formatErrorMessage(error, 'The jobs history could not be loaded.')}
        />
      ) : null}
      <div className="tab-content">
        <div className="diagnostics-header">
          <p className="muted">{jobs.length} jobs</p>
          <button
            className="ghost-button"
            type="button"
            onClick={onRefresh}
            disabled={loading}
          >
            Refresh
          </button>
        </div>

        <div className="jobs-list">
          {loading ? (
            <p className="muted">Loading jobs...</p>
          ) : jobs.length === 0 ? (
            <p className="muted">No jobs recorded yet.</p>
          ) : (
            <table className="jobs-table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Status</th>
                  <th>Started</th>
                  <th>Duration</th>
                  <th>Summary</th>
                </tr>
              </thead>
              <tbody>
                {jobs.map((job) => (
                  <tr key={job.id} className={`status-${job.status}`}>
                    <td className="job-type">{job.type}</td>
                    <td>
                      <span className={`job-badge status-${job.status}`}>
                        {job.status.toUpperCase()}
                      </span>
                    </td>
                    <td>{new Date(job.started_at).toLocaleString()}</td>
                    <td>
                      {job.finished_at
                        ? formatDuration(
                            new Date(job.finished_at).getTime() -
                              new Date(job.started_at).getTime(),
                          )
                        : '—'}
                    </td>
                    <td>
                      <span className="job-summary">
                        {job.summary || job.error || '—'}
                      </span>
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
