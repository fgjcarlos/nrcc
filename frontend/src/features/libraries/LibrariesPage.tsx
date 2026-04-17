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
          <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Extensions</p>
          <h2 className="page-title text-3xl mt-1">Libraries</h2>
          <p className="mt-2 max-w-2xl text-sm text-base-content/65">
            Install and manage runtime packages without leaving the old-style operations console.
          </p>
        </div>
        <div className="flex flex-wrap gap-2 text-sm text-base-content/70">
          <span className="rounded-full bg-base-300/60 px-3 py-1">Installed: {libraries?.items.length ?? 0}</span>
          <span className="rounded-full bg-base-300/60 px-3 py-1">Busy: {busy ? 'Yes' : 'No'}</span>
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

      <article className="surface-card border border-base-300/60 p-6 md:p-7">
        <div className="mb-5">
          <h3 className="section-title">Installed packages</h3>
          <p className="mt-1 text-sm text-base-content/60">Packages currently available inside the runtime.</p>
        </div>
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
      </article>
    </>
  )
}
