import { useState } from 'react'
import type { DoctorReport, LogEntry, JobRecord } from '../../api'
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
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Support</p>
          <h2 className="text-3xl font-bold tracking-tight text-base-content mt-1">Diagnostics</h2>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Doctor checks, support logs, and job history in a single operational workspace.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            className="action-btn-primary"
            type="button"
            onClick={onExport}
            disabled={exporting}
          >
            {exporting ? 'Exporting...' : 'Export Support Bundle'}
          </button>
        </div>
      </header>

      <article className="surface-card border border-base-300/60 p-6 md:p-7">
        <div>
          <div className="surface-panel section-tabbar border border-base-300/60 mb-6">
            <button
              className={`section-tab ${activeTab === 'doctor' ? 'section-tab-active' : ''}`}
              type="button"
              onClick={() => setActiveTab('doctor')}
            >
              Doctor
            </button>
            <button
              className={`section-tab ${activeTab === 'logs' ? 'section-tab-active' : ''}`}
              type="button"
              onClick={() => setActiveTab('logs')}
            >
              Logs
            </button>
            <button
              className={`section-tab ${activeTab === 'jobs' ? 'section-tab-active' : ''}`}
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
