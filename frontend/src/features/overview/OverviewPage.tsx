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
    <>
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-8">
        <div>
          <p className="text-sm font-semibold text-base-content/70 uppercase tracking-wide">Runtime</p>
          <h2 className="text-3xl font-bold text-base-content mt-1">Dashboard</h2>
        </div>
        <div className="flex gap-2">
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
      </header>

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
    </>
  )
}

export { buildGlobalStatus }
