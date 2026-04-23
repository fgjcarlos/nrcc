import { useState } from 'react'
import { motion } from 'framer-motion'
import { Link } from 'react-router-dom'

import { AnimatedCard, InlineNotice, StatCard } from '../../common/components'
import { formatErrorMessage, formatUptime } from '../../common/utils/format'
import { RuntimeDetailsPanel } from './RuntimeDetailsPanel'
import { SystemInfoPanel } from './SystemInfoPanel'
import { RestartButton } from './RestartButton'
import { useOverviewData } from './useOverviewData'

function formatTimestamp(value?: string) {
  if (!value) {
    return 'Unavailable'
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleString()
}

function getLatestBackup(backups: any) {
  return backups?.items.reduce((latest: any, current: any) => {
    if (!latest) {
      return current
    }

    return new Date(current.createdAt).getTime() > new Date(latest.createdAt).getTime() ? current : latest
  }, undefined)
}

export function OverviewPage() {
  const { runtime, runtimeLoading, runtimeError, systemInfo, systemLoading, systemError, backups, backupsLoading, environment, environmentLoading, operationStatus, globalStatus, restarting, onRestart } = useOverviewData()
  const [confirmRestart, setConfirmRestart] = useState(false)
  const pageError = runtimeError ?? systemError
  const latestBackup = getLatestBackup(backups)
  const runtimeState = runtimeLoading ? 'Checking' : runtime?.running ? 'Running' : 'Stopped'
  const runtimeHealth = runtimeLoading ? 'Checking' : runtime?.healthy ? 'Healthy' : 'Needs attention'
  const runtimeTone = runtimeLoading ? 'info' : runtime?.running ? (runtime?.healthy ? 'ok' : 'warn') : 'error'
  const operationTone = operationStatus?.busy ? 'warn' : latestBackup ? 'ok' : 'neutral'
  const activityItems = [
    latestBackup
      ? {
          label: 'Backup created',
          detail: latestBackup.reason,
          timestamp: latestBackup.createdAt,
        }
      : null,
    systemInfo?.timestamp
      ? {
          label: 'System snapshot refreshed',
          detail: `${systemInfo.hostname} • ${systemInfo.goos}/${systemInfo.goarch}`,
          timestamp: systemInfo.timestamp,
        }
      : null,
    runtime?.startedAt
      ? {
          label: 'Runtime started',
          detail: runtime.version ? `Node-RED ${runtime.version}` : 'Primary runtime online',
          timestamp: runtime.startedAt,
        }
      : null,
    systemInfo?.localAccess?.url
      ? {
          label: 'Preferred local access',
          detail: systemInfo.localAccess.url,
          timestamp: systemInfo.timestamp,
        }
      : null,
    runtime?.lastExit
      ? {
          label: 'Last runtime exit',
          detail: runtime.lastError || 'Runtime reported a previous stop event.',
          timestamp: runtime.lastExit,
        }
      : null,
  ]
    .filter((item): item is { label: string; detail: string; timestamp: string } => Boolean(item))
    .sort((left, right) => new Date(right.timestamp).getTime() - new Date(left.timestamp).getTime())

  const handleRestartConfirm = () => {
    setConfirmRestart(false)
    onRestart()
  }

  return (
    <div className="space-y-6 sm:space-y-8">
      <motion.section
        className="surface-panel relative overflow-hidden border border-base-300 px-4 py-6 sm:px-8 sm:py-8"
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
      >
        <div className="absolute left-0 right-0 top-0 h-1 bg-gradient-to-r from-primary via-primary/50 to-transparent"></div>

        <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-3xl">
            <p className="text-xs font-semibold uppercase tracking-[0.32em] text-base-content/55">System overview</p>
            <h2 className="page-title mt-3 sm:text-5xl">Dashboard</h2>
            <p className="mt-4 text-base text-base-content/70 sm:text-lg">
              Clear runtime, configuration, and operations zones for the main Node-RED workspace.
            </p>
          </div>
          <div className="flex w-full flex-col items-start gap-3 sm:w-auto sm:items-end">
            <span
              className={`self-start rounded-full px-4 py-2 text-xs font-semibold uppercase tracking-[0.18em] sm:self-auto ${
                globalStatus.tone === 'ok'
                  ? 'bg-success/10 text-success'
                  : globalStatus.tone === 'warn'
                    ? 'bg-warning/10 text-warning'
                    : 'bg-info/10 text-info'
              }`}
            >
              {globalStatus.title}
            </span>
            <RestartButton
              confirmRestart={confirmRestart}
              blocked={operationStatus?.busy ?? false}
              restarting={restarting}
              onConfirm={handleRestartConfirm}
              onCancel={() => setConfirmRestart(false)}
              onRequest={() => setConfirmRestart(true)}
            />
            <div className="flex w-full flex-col gap-2 text-sm sm:w-auto sm:flex-row sm:flex-wrap sm:justify-end">
              <Link className="action-btn-secondary w-full sm:w-auto" to="/app/logs">
                View logs
              </Link>
              <Link className="action-btn-ghost w-full sm:w-auto" to="/app/backups">
                Backups
              </Link>
            </div>
          </div>
        </div>
      </motion.section>

      <motion.nav
        className="glass-panel sticky top-16 z-10 border px-2 py-2 sm:top-20 sm:px-3 sm:py-3"
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25, delay: 0.05 }}
        aria-label="Dashboard sections"
      >
        <div className="-mx-1 flex snap-x snap-mandatory gap-2 overflow-x-auto px-1 pb-1 sm:mx-0 sm:flex-wrap sm:overflow-visible sm:px-0 sm:pb-0">
          <a className="action-btn-secondary shrink-0 snap-start" href="#runtime-zone">
            Runtime zone
          </a>
          <a className="action-btn-secondary shrink-0 snap-start" href="#configuration-zone">
            Configuration zone
          </a>
          <a className="action-btn-secondary shrink-0 snap-start" href="#operations-zone">
            Operations zone
          </a>
        </div>
      </motion.nav>

      {confirmRestart ? (
        <motion.div
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: -8 }}
          transition={{ duration: 0.2 }}
        >
          <InlineNotice
            tone="warn"
            title="Confirm runtime restart"
            detail="Node-RED will be stopped and started again. Status and logs will refresh automatically."
          />
        </motion.div>
      ) : null}

      {pageError ? (
        <motion.div
          initial={{ opacity: 0, y: -8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.2 }}
        >
          <InlineNotice
            tone="error"
            title="System information is incomplete"
            detail={formatErrorMessage(pageError, 'The dashboard could not refresh all runtime details.')}
          />
        </motion.div>
      ) : null}

      <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <AnimatedCard delay={0.1}>
          <StatCard
            label="Managed instances"
            value={runtimeLoading ? 'Checking' : runtime?.running ? '1 / 1' : '0 / 1'}
            detail={runtime?.running ? 'Primary runtime available for actions.' : 'Primary runtime currently offline.'}
            accent={runtime?.running ? 'ok' : 'error'}
          />
        </AnimatedCard>
        <AnimatedCard delay={0.15}>
          <StatCard
            label="Health"
            value={runtimeHealth}
            detail={runtime?.healthy ? 'Checks are passing for the runtime service.' : 'Review notices, logs, and diagnostics.'}
            accent={runtimeLoading ? 'info' : runtime?.healthy ? 'ok' : 'warn'}
          />
        </AnimatedCard>
        <AnimatedCard delay={0.2}>
          <StatCard
            label="System status"
            value={globalStatus.title}
            detail={globalStatus.detail}
            accent={globalStatus.tone === 'ok' ? 'ok' : globalStatus.tone === 'warn' ? 'warn' : 'neutral'}
          />
        </AnimatedCard>
        <AnimatedCard delay={0.25}>
          <StatCard
            label="Last backup"
            value={backupsLoading ? 'Checking' : latestBackup ? formatTimestamp(latestBackup.createdAt) : 'Not created'}
            detail={latestBackup ? latestBackup.reason : 'Create a recovery point from the backups workspace.'}
            accent={latestBackup ? 'info' : 'neutral'}
          />
        </AnimatedCard>
      </section>

      <motion.section
        id="runtime-zone"
        className="space-y-6 scroll-mt-28"
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.35 }}
      >
        <div className="flex flex-col gap-2 px-1 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/50">Zone 01</p>
            <h2 className="section-title text-2xl">Runtime status</h2>
            <p className="mt-2 max-w-3xl text-sm text-base-content/65">
              Keep the managed Node-RED instance front and center with live state, operator controls, and runtime details.
            </p>
            {systemInfo?.localAccess ? (
              <p className="mt-3 text-sm text-base-content/65">
                Preferred local access: <span className="font-medium text-base-content">{systemInfo.localAccess.url}</span>
              </p>
            ) : null}
          </div>
          <span
            className={`badge badge-lg ${
              runtimeTone === 'ok'
                ? 'badge-success'
                : runtimeTone === 'warn'
                  ? 'badge-warning'
                  : runtimeTone === 'error'
                    ? 'badge-error'
                    : 'badge-info'
            }`}
          >
            {runtimeState}
          </span>
        </div>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-[minmax(0,1.15fr)_minmax(280px,0.85fr)] xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.85fr)]">
          <AnimatedCard delay={0.4} className="h-full">
            <article
              className={`surface-card section-card h-full border p-6 sm:p-7 ${
                runtimeTone === 'ok'
                  ? 'section-card--success'
                  : runtimeTone === 'warn'
                    ? 'section-card--warning'
                    : runtimeTone === 'error'
                      ? 'section-card--error'
                      : 'section-card--info'
              }`}
            >
              <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
                <div className="max-w-2xl">
                  <div className="flex flex-wrap items-center gap-3">
                    <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/55">Primary runtime</p>
                    <span
                      className={`badge ${
                        runtimeTone === 'ok'
                          ? 'badge-success'
                          : runtimeTone === 'warn'
                            ? 'badge-warning'
                            : runtimeTone === 'error'
                              ? 'badge-error'
                              : 'badge-info'
                      }`}
                    >
                      {runtimeState}
                    </span>
                  </div>
                  <h3 className="mt-3 text-3xl font-semibold tracking-tight text-base-content">
                    {runtime?.version ? `Node-RED ${runtime.version}` : 'Node-RED service'}
                  </h3>
                  <p className="mt-3 text-sm text-base-content/70 sm:text-base">
                    Restart the runtime, jump directly into logs, and watch the service footprint without leaving the dashboard.
                  </p>
                </div>
                <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap">
                  <RestartButton
                    confirmRestart={confirmRestart}
                    blocked={operationStatus?.busy ?? false}
                    restarting={restarting}
                    onConfirm={handleRestartConfirm}
                    onCancel={() => setConfirmRestart(false)}
                    onRequest={() => setConfirmRestart(true)}
                  />
                  <Link className="action-btn-secondary w-full sm:w-auto" to="/app/logs">
                    Open logs
                  </Link>
                  <Link className="action-btn-ghost w-full sm:w-auto" to="/app/diagnostics">
                    Diagnostics
                  </Link>
                </div>
              </div>
              <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                <div className="rounded-2xl border border-base-300/60 bg-base-200/30 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">Uptime</p>
                  <p className="mt-2 text-lg font-semibold text-base-content">
                    {runtimeLoading ? 'Loading...' : formatUptime(runtime?.uptimeSec ?? 0)}
                  </p>
                </div>
                <div className="rounded-2xl border border-base-300/60 bg-base-200/30 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">Port</p>
                  <p className="mt-2 text-lg font-semibold text-base-content">{runtime?.port ? String(runtime.port) : 'N/A'}</p>
                </div>
                <div className="rounded-2xl border border-base-300/60 bg-base-200/30 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">PID</p>
                  <p className="mt-2 text-lg font-semibold text-base-content">{runtime?.pid ? String(runtime.pid) : 'N/A'}</p>
                </div>
                <div className="rounded-2xl border border-base-300/60 bg-base-200/30 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">Last start</p>
                  <p className="mt-2 text-lg font-semibold text-base-content">{formatTimestamp(runtime?.startedAt)}</p>
                </div>
              </div>
            </article>
          </AnimatedCard>
          <AnimatedCard delay={0.45} className="h-full">
            <RuntimeDetailsPanel runtime={runtime} runtimeLoading={runtimeLoading} />
          </AnimatedCard>
        </div>
      </motion.section>

      <motion.section
        id="configuration-zone"
        className="space-y-6 scroll-mt-28"
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.45 }}
      >
        <div className="flex flex-col gap-2 px-1 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/50">Zone 02</p>
            <h2 className="section-title text-2xl">Configuration and environment</h2>
            <p className="mt-2 max-w-3xl text-sm text-base-content/65">
              Separate host context from mutable runtime settings so operators can jump to configuration work faster.
            </p>
          </div>
          <span className="rounded-full bg-base-300/60 px-3 py-1 text-xs font-medium text-base-content/70">
            Env vars: {environmentLoading ? 'Loading...' : environment?.variables.length ?? 0}
          </span>
        </div>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-[minmax(0,1fr)_minmax(280px,0.9fr)] xl:grid-cols-[minmax(0,1fr)_minmax(320px,0.9fr)]">
          <AnimatedCard delay={0.5} className="h-full">
            <SystemInfoPanel systemInfo={systemInfo} systemLoading={systemLoading} />
          </AnimatedCard>
          <AnimatedCard delay={0.55} className="h-full">
            <article className="surface-card section-card section-card--default h-full border p-6">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-primary"></div>
                <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/55">Configuration</p>
              </div>
              <h3 className="mt-3 text-xl font-semibold text-base-content">Workspace snapshot</h3>
              <p className="mt-2 text-sm text-base-content/68">
                The most important runtime settings and file locations stay visible without turning the dashboard into another long settings page.
              </p>
              <div className="mt-6 space-y-3">
                <div className="flex flex-col items-start justify-between gap-2 rounded-2xl border border-base-300/60 bg-base-200/30 px-4 py-3 sm:flex-row sm:items-center sm:gap-4">
                  <span className="text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">Managed env vars</span>
                  <span className="text-sm font-semibold text-base-content">{environmentLoading ? 'Loading...' : environment?.variables.length ?? 0}</span>
                </div>
                <div className="flex flex-col items-start justify-between gap-2 rounded-2xl border border-base-300/60 bg-base-200/30 px-4 py-3 sm:flex-row sm:items-center sm:gap-4">
                  <span className="text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">Data directory</span>
                  <span className="max-w-full break-all text-sm font-semibold text-base-content sm:max-w-[16rem] sm:truncate">{runtime?.dataDir || 'N/A'}</span>
                </div>
                <div className="flex flex-col items-start justify-between gap-2 rounded-2xl border border-base-300/60 bg-base-200/30 px-4 py-3 sm:flex-row sm:items-center sm:gap-4">
                  <span className="text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">Host refresh</span>
                  <span className="text-sm font-semibold text-base-content">{formatTimestamp(systemInfo?.timestamp)}</span>
                </div>
              </div>
              <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:flex-wrap">
                <Link className="action-btn-primary w-full sm:w-auto" to="/app/config">
                  Open config
                </Link>
                <Link className="action-btn-secondary w-full sm:w-auto" to="/app/environment">
                  Manage environment
                </Link>
              </div>
            </article>
          </AnimatedCard>
        </div>
      </motion.section>

      <motion.section
        id="operations-zone"
        className="space-y-6 scroll-mt-28"
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, delay: 0.55 }}
      >
        <div className="flex flex-col gap-2 px-1 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/50">Zone 03</p>
            <h2 className="section-title text-2xl">Operations and recent activity</h2>
            <p className="mt-2 max-w-3xl text-sm text-base-content/65">
              Highlight the latest backup, in-progress work, and activity history so maintenance tasks stay visible instead of buried lower on the page.
            </p>
          </div>
          <span
            className={`rounded-full px-3 py-1 text-xs font-medium ${
              operationTone === 'ok'
                ? 'bg-success/10 text-success'
                : operationTone === 'warn'
                  ? 'bg-warning/10 text-warning'
                  : 'bg-base-300/70 text-base-content/70'
            }`}
          >
            {operationStatus?.busy ? 'Operation in progress' : latestBackup ? 'Recovery ready' : 'No backup yet'}
          </span>
        </div>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-[minmax(0,1.1fr)_minmax(280px,0.9fr)] xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]">
          <AnimatedCard delay={0.6} className="h-full">
            <article className="surface-card section-card section-card--info h-full border p-6">
              <div className="flex flex-col items-start justify-between gap-3 sm:flex-row sm:items-center">
                <div>
                  <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/55">Timeline</p>
                  <h3 className="mt-2 text-xl font-semibold text-base-content">Recent activity</h3>
                </div>
                <Link className="action-btn-ghost w-full sm:w-auto" to="/app/logs">
                  Open logs
                </Link>
              </div>
              {activityItems.length ? (
                <div className="mt-6 space-y-4">
                  {activityItems.map((item, index) => (
                    <div key={`${item.label}-${item.timestamp}`} className="flex gap-4 rounded-2xl border border-base-300/60 bg-base-200/20 p-4">
                      <div className="flex flex-col items-center">
                        <div className="mt-1 h-2.5 w-2.5 rounded-full bg-info"></div>
                        {index < activityItems.length - 1 ? <div className="mt-2 h-full w-px bg-base-300/70"></div> : null}
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                          <p className="font-semibold text-base-content">{item.label}</p>
                          <span className="text-xs uppercase tracking-[0.16em] text-base-content/50">{formatTimestamp(item.timestamp)}</span>
                        </div>
                        <p className="mt-1 text-sm text-base-content/68">{item.detail}</p>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="mt-6 text-sm text-base-content/60">No recent activity has been recorded yet.</p>
              )}
            </article>
          </AnimatedCard>
          <div className="grid gap-6">
            <AnimatedCard delay={0.65} className="h-full">
              <article
                className={`surface-card section-card h-full border p-6 ${
                  latestBackup ? 'section-card--success' : 'section-card--warning'
                }`}
              >
                <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/55">Recovery</p>
                <h3 className="mt-2 text-xl font-semibold text-base-content">Backup status</h3>
                <p className="mt-2 text-sm text-base-content/68">
                  {latestBackup
                    ? `Latest archive ${latestBackup.archiveName} is available for restore.`
                    : 'No backup has been created yet. Generate one before making high-risk changes.'}
                </p>
                <div className="mt-5 rounded-2xl border border-base-300/60 bg-base-200/20 p-4">
                  <p className="text-xs font-semibold uppercase tracking-[0.18em] text-base-content/50">Last backup</p>
                  <p className="mt-2 text-lg font-semibold text-base-content">
                    {backupsLoading ? 'Loading...' : latestBackup ? formatTimestamp(latestBackup.createdAt) : 'Not created'}
                  </p>
                </div>
                <div className="mt-5 flex flex-col gap-2 sm:flex-row sm:flex-wrap">
                  <Link className="action-btn-primary w-full sm:w-auto" to="/app/backups">
                    Open backups
                  </Link>
                </div>
              </article>
            </AnimatedCard>
            <AnimatedCard delay={0.7} className="h-full">
              <article
                className={`surface-card section-card h-full border p-6 ${
                  operationStatus?.busy ? 'section-card--warning' : 'section-card--default'
                }`}
              >
                <p className="text-xs font-semibold uppercase tracking-[0.24em] text-base-content/55">Quick actions</p>
                <h3 className="mt-2 text-xl font-semibold text-base-content">Maintenance lanes</h3>
                <p className="mt-2 text-sm text-base-content/68">
                  {operationStatus?.busy
                    ? `${operationStatus.type ?? 'Operation'} in progress${operationStatus.detail ? `: ${operationStatus.detail}` : '.'}`
                    : 'Jump into diagnostics, updates, or library management from a dedicated operations zone.'}
                </p>
                <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <Link className="action-btn-secondary w-full" to="/app/diagnostics">
                    Diagnostics
                  </Link>
                  <Link className="action-btn-secondary w-full" to="/app/updates">
                    Updates
                  </Link>
                  <Link className="action-btn-secondary w-full" to="/app/libraries">
                    Libraries
                  </Link>
                  <Link className="action-btn-secondary w-full" to="/app/logs">
                    Runtime logs
                  </Link>
                </div>
              </article>
            </AnimatedCard>
          </div>
        </div>
      </motion.section>
    </div>
  )
}
