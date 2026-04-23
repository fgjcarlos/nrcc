import { KeyboardEvent, useRef, useState } from 'react'
import { DoctorTab } from './DoctorTab'
import { LogsTab } from './LogsTab'
import { JobsTab } from './JobsTab'
import { useDiagnosticsData } from './useDiagnosticsData'

export function DiagnosticsPage() {
  const { report, reportLoading, reportError, logs, logsLoading, logsError, jobs, jobsLoading, jobsError, exporting, onRefreshReport, onRefreshLogs, onRefreshJobs, onExport } = useDiagnosticsData()
  const tabs = [
    { id: 'doctor', label: 'Doctor' },
    { id: 'logs', label: 'Logs' },
    { id: 'jobs', label: 'Jobs' },
  ] as const
  const [activeTab, setActiveTab] = useState<'doctor' | 'logs' | 'jobs'>('doctor')
  const tabRefs = useRef<Record<(typeof tabs)[number]['id'], HTMLButtonElement | null>>({
    doctor: null,
    logs: null,
    jobs: null,
  })

  function activateTab(nextTab: (typeof tabs)[number]['id']) {
    setActiveTab(nextTab)
    tabRefs.current[nextTab]?.focus()
  }

  function handleTabKeyDown(event: KeyboardEvent<HTMLButtonElement>, currentIndex: number) {
    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
      event.preventDefault()
      activateTab(tabs[(currentIndex + 1) % tabs.length].id)
    }

    if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
      event.preventDefault()
      activateTab(tabs[(currentIndex - 1 + tabs.length) % tabs.length].id)
    }

    if (event.key === 'Home') {
      event.preventDefault()
      activateTab(tabs[0].id)
    }

    if (event.key === 'End') {
      event.preventDefault()
      activateTab(tabs[tabs.length - 1].id)
    }
  }

  return (
    <>
      <header className="mb-8 flex flex-col gap-6 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Support</p>
          <h2 className="page-title text-3xl mt-1">Diagnostics</h2>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Doctor checks, support logs, and job history in a single operational workspace.
          </p>
        </div>
        <div className="flex w-full gap-2 sm:w-auto">
          <button
            className="action-btn-primary w-full sm:w-auto"
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
          <div className="surface-panel section-tabbar border border-base-300/60 mb-6" role="tablist" aria-label="Diagnostics views">
            {tabs.map((tab, index) => {
              const isActive = activeTab === tab.id

              return (
                <button
                  key={tab.id}
                  ref={(node) => {
                    tabRefs.current[tab.id] = node
                  }}
                  id={`diagnostics-tab-${tab.id}`}
                  className={`section-tab ${isActive ? 'section-tab-active' : ''}`}
                  type="button"
                  role="tab"
                  tabIndex={isActive ? 0 : -1}
                  aria-selected={isActive}
                  aria-controls={`diagnostics-panel-${tab.id}`}
                  onClick={() => setActiveTab(tab.id)}
                  onKeyDown={(event) => handleTabKeyDown(event, index)}
                >
                  {tab.label}
                </button>
              )
            })}
          </div>

          {activeTab === 'doctor' && (
            <div id="diagnostics-panel-doctor" role="tabpanel" aria-labelledby="diagnostics-tab-doctor">
              <DoctorTab
                report={report}
                loading={reportLoading}
                error={reportError}
                onRefresh={onRefreshReport}
              />
            </div>
          )}

          {activeTab === 'logs' && (
            <div id="diagnostics-panel-logs" role="tabpanel" aria-labelledby="diagnostics-tab-logs">
              <LogsTab
                logs={logs}
                loading={logsLoading}
                error={logsError}
                onRefresh={onRefreshLogs}
              />
            </div>
          )}

          {activeTab === 'jobs' && (
            <div id="diagnostics-panel-jobs" role="tabpanel" aria-labelledby="diagnostics-tab-jobs">
              <JobsTab
                jobs={jobs}
                loading={jobsLoading}
                error={jobsError}
                onRefresh={onRefreshJobs}
              />
            </div>
          )}
        </div>
      </article>
    </>
  )
}
