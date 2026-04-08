import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api, type LibraryList, type OperationStatus } from '../../api'
import { InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import type { ToastTone } from '../../common/types'
import { LibraryCard } from './LibraryCard'
import { InstallForm } from './InstallForm'

export function LibrariesPage({
  libraries,
  loading,
  error,
  operationStatus,
  onChanged,
}: {
  libraries?: LibraryList
  loading: boolean
  error: unknown
  operationStatus?: OperationStatus
  onChanged: (message: string, tone: ToastTone) => Promise<void>
}) {
  const [packageName, setPackageName] = useState('')

  const installMutation = useMutation({
    mutationFn: api.installLibrary,
    onSuccess: async (result) => {
      setPackageName('')
      await onChanged(result.message, 'success')
    },
    onError: async (mutationError) => {
      await onChanged(formatErrorMessage(mutationError, 'The package could not be installed.'), 'error')
    },
  })

  const uninstallMutation = useMutation({
    mutationFn: api.uninstallLibrary,
    onSuccess: async (result) => {
      await onChanged(result.message, 'success')
    },
    onError: async (mutationError) => {
      await onChanged(formatErrorMessage(mutationError, 'The package could not be removed.'), 'error')
    },
  })

  const busy = operationStatus?.busy ?? false

  return (
    <>
      <header className="topbar">
        <div>
          <p className="eyebrow">Runtime</p>
          <h2>Libraries</h2>
        </div>
      </header>

      {error ? (
        <InlineNotice
          tone="error"
          title="Libraries unavailable"
          detail={formatErrorMessage(error, 'Installed packages could not be loaded.')}
        />
      ) : null}

      {busy ? (
        <InlineNotice
          tone="warn"
          title="System busy"
          detail={
            (operationStatus?.type ? `${operationStatus.type} in progress` : 'Another operation is in progress') +
            (operationStatus?.detail ? `: ${operationStatus.detail}` : '.')
          }
        />
      ) : null}

      <InstallForm
        packageName={packageName}
        busy={busy}
        isPending={installMutation.isPending}
        onChange={setPackageName}
        onSubmit={() => installMutation.mutate(packageName)}
      />

      <article className="panel">
        <div className="panel-header">
          <h3>Installed packages</h3>
        </div>
        {loading ? <p className="muted">Loading installed packages...</p> : null}
        {!loading && (!libraries || libraries.items.length === 0) ? <p className="muted">No additional packages installed.</p> : null}
        {libraries?.items.length ? (
          <div className="library-list">
            {libraries.items.map((item) => (
              <LibraryCard
                key={item.name}
                item={item}
                isPending={uninstallMutation.isPending}
                busy={busy}
                onUninstall={() => uninstallMutation.mutate(item.name)}
              />
            ))}
          </div>
        ) : null}
      </article>
    </>
  )
}
