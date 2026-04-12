import { useState } from 'react'
import type { DoctorReport, LogEntry, JobRecord } from '../../api'
import { InlineNotice } from '../../common/components'
import { DoctorTab } from './DoctorTab'
import { LogsTab } from './LogsTab'
import { JobsTab } from './JobsTab'

export function DiagnosticsPage({
  report,
  reportLoading,
  reportError,
  logs,
  logsLoading,
  logsError,
  jobs,
  jobsLoading,
  jobsError,
  exporting,
  onRefreshReport,
  onRefreshLogs,
  onRefreshJobs,
  onExport,
}: {
  report?: DoctorReport
  reportLoading: boolean
  reportError: unknown
  logs: LogEntry[]
  logsLoading: boolean
  logsError: unknown
  jobs: JobRecord[]
  jobsLoading: boolean
  jobsError: unknown
  exporting: boolean
  onRefreshReport: () => Promise<void>
  onRefreshLogs: () => Promise<void>
  onRefreshJobs: () => Promise<void>
  onExport: () => void
}) {
  const [activeTab, setActiveTab] = useState<'doctor' | 'logs' | 'jobs'>('doctor')

  return (
    <>
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-8">
        <div>
          <p className="text-sm font-semibold text-base-content/70 uppercase tracking-wide">Support</p>
          <h2 className="text-3xl font-bold text-base-content mt-1">Diagnostics</h2>
        </div>
        <div className="flex gap-2">
          <button
            className="btn btn-primary"
            type="button"
            onClick={onExport}
            disabled={exporting}
          >
            {exporting ? 'Exporting...' : 'Export Support Bundle'}
          </button>
        </div>
      </header>

      <article className="card bg-base-200 shadow-elevation-2 rounded-lg">
        <div className="card-body">
          <div className="tabs tabs-bordered mb-6">
            <button
              className={`tab ${activeTab === 'doctor' ? 'tab-active' : ''}`}
              type="button"
              onClick={() => setActiveTab('doctor')}
            >
              Doctor
            </button>
            <button
              className={`tab ${activeTab === 'logs' ? 'tab-active' : ''}`}
              type="button"
              onClick={() => setActiveTab('logs')}
            >
              Logs
            </button>
            <button
              className={`tab ${activeTab === 'jobs' ? 'tab-active' : ''}`}
              type="button"
              onClick={() => setActiveTab('jobs')}
            >
              Jobs
            </button>
          </div>

          {activeTab === 'doctor' && (
            <DoctorTab
              report={report}
              loading={reportLoading}
              error={reportError}
              onRefresh={onRefreshReport}
            />
          )}

          {activeTab === 'logs' && (
            <LogsTab
              logs={logs}
              loading={logsLoading}
              error={logsError}
              onRefresh={onRefreshLogs}
            />
          )}

          {activeTab === 'jobs' && (
            <JobsTab
              jobs={jobs}
              loading={jobsLoading}
              error={jobsError}
              onRefresh={onRefreshJobs}
            />
          )}
        </div>
      </article>
    </>
  )
}
