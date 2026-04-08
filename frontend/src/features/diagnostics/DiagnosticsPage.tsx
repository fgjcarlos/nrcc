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
      <header className="topbar">
        <div>
          <p className="eyebrow">Support</p>
          <h2>Diagnostics</h2>
        </div>
        <div className="topbar-actions">
          <button
            className="primary-button"
            type="button"
            onClick={onExport}
            disabled={exporting}
          >
            {exporting ? 'Exporting...' : 'Export Support Bundle'}
          </button>
        </div>
      </header>

      <article className="panel diagnostics-panel">
        <div className="panel-header diagnostics-tabs">
          <button
            className={activeTab === 'doctor' ? 'tab-button active' : 'tab-button'}
            type="button"
            onClick={() => setActiveTab('doctor')}
          >
            Doctor
          </button>
          <button
            className={activeTab === 'logs' ? 'tab-button active' : 'tab-button'}
            type="button"
            onClick={() => setActiveTab('logs')}
          >
            Logs
          </button>
          <button
            className={activeTab === 'jobs' ? 'tab-button active' : 'tab-button'}
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
      </article>
    </>
  )
}
