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
      <header className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-6 mb-8">
        <div>
          <p className="text-sm font-semibold text-base-content/70 uppercase tracking-wide">Runtime</p>
          <h2 className="text-3xl font-bold text-base-content mt-1">Libraries</h2>
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

      <article className="card bg-base-200">
        <div className="card-body">
          <h3 className="card-title text-2xl">Installed packages</h3>
          {loading ? <p className="text-sm text-base-content/60">Loading installed packages...</p> : null}
          {!loading && (!libraries || libraries.items.length === 0) ? <p className="text-sm text-base-content/60">No additional packages installed.</p> : null}
          {libraries?.items.length ? (
            <div className="space-y-4">
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
        </div>
      </article>
    </>
  )
}
