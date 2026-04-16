import { useState } from 'react'

import type { RuntimeStatus, SystemInfo } from '../../api'
import { StatCard, InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import { buildGlobalStatus } from '../../common/utils/status'
import type { GlobalStatus } from '../../common/types'
import { RuntimeDetailsPanel } from './RuntimeDetailsPanel'
import { SystemInfoPanel } from './SystemInfoPanel'
import { RestartButton } from './RestartButton'

export function OverviewPage({
  runtime,
  runtimeLoading,
  runtimeError,
  systemInfo,
  systemLoading,
  systemError,
  restarting,
  onRestart,
  globalStatus,
}: {
  runtime?: RuntimeStatus
  runtimeLoading: boolean
  runtimeError: unknown
  systemInfo?: SystemInfo
  systemLoading: boolean
  systemError: unknown
  restarting: boolean
  onRestart: () => void
  globalStatus: GlobalStatus
}) {
  const [confirmRestart, setConfirmRestart] = useState(false)
  const pageError = runtimeError ?? systemError

  return (
    <div className="space-y-8">
      <section className="surface-panel border border-base-300 px-6 py-8 sm:px-8">
        <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
          <div className="max-w-3xl">
            <p className="text-xs uppercase tracking-[0.32em] text-base-content/55">System overview</p>
            <h2 className="mt-3 text-4xl font-bold tracking-tight text-base-content sm:text-5xl">Dashboard</h2>
            <p className="mt-4 text-base text-base-content/70 sm:text-lg">
              Real-time control for the local Node-RED runtime, system health, and operational state.
            </p>
          </div>
          <div className="flex flex-col items-start gap-3 sm:items-end">
            <span className="rounded-full bg-base-300/60 px-4 py-2 text-xs font-medium uppercase tracking-[0.18em] text-base-content/70">
              {globalStatus.title}
            </span>
            <RestartButton
              confirmRestart={confirmRestart}
              restarting={restarting}
              onConfirm={() => {
                setConfirmRestart(false)
                onRestart()
              }}
              onCancel={() => setConfirmRestart(false)}
              onRequest={() => setConfirmRestart(true)}
            />
          </div>
        </div>
      </section>

      {confirmRestart ? (
        <InlineNotice
          tone="warn"
          title="Confirm runtime restart"
          detail="Node-RED will be stopped and started again. Status and logs will refresh automatically."
        />
      ) : null}

      {pageError ? (
        <InlineNotice
          tone="error"
          title="System information is incomplete"
          detail={formatErrorMessage(pageError, 'The dashboard could not refresh all runtime details.')}
        />
      ) : null}

      <section className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <StatCard
          label="Runtime state"
          value={runtimeLoading ? 'Loading...' : runtime?.running ? 'Running' : 'Stopped'}
          accent={runtime?.running ? 'ok' : 'warn'}
        />
        <StatCard
          label="Health"
          value={runtimeLoading ? 'Loading...' : runtime?.healthy ? 'Healthy' : 'Unavailable'}
          accent={runtime?.healthy ? 'ok' : 'warn'}
        />
        <StatCard
          label="Version"
          value={runtimeLoading ? 'Loading...' : runtime?.version || 'Unknown'}
          accent="neutral"
        />
        <StatCard label="Global status" value={globalStatus.title} accent={globalStatus.tone} />
      </section>

      <section className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <RuntimeDetailsPanel runtime={runtime} runtimeLoading={runtimeLoading} />
        <SystemInfoPanel systemInfo={systemInfo} systemLoading={systemLoading} />
      </section>
    </div>
  )
}

export { buildGlobalStatus }
