import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import type { UpdateStatus, OperationStatus } from '../../api'
import { api } from '../../api'
import { Detail, InlineNotice } from '../../common/components'
import { formatErrorMessage } from '../../common/utils/format'
import type { ToastTone } from '../../common/types'

export function UpdateCard({
  updateStatus,
  operationStatus,
  onChanged,
}: {
  updateStatus: UpdateStatus
  operationStatus?: OperationStatus
  onChanged: (message: string, tone: ToastTone) => Promise<void>
}) {
  const [confirmUpdate, setConfirmUpdate] = useState(false)

  const applyMutation = useMutation({
    mutationFn: api.applyUpdate,
    onSuccess: async (result) => {
      setConfirmUpdate(false)
      await onChanged(result.message, result.rolledBack ? 'error' : 'success')
    },
    onError: async (mutationError) => {
      setConfirmUpdate(false)
      await onChanged(formatErrorMessage(mutationError, 'The update could not be applied.'), 'error')
    },
  })

  const busy = operationStatus?.busy ?? false

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-3">
        <div className="surface-panel border border-base-300/60 p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/50">Installed</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{updateStatus.installedVersion || 'Unknown'}</div>
        </div>
        <div className="surface-panel border border-base-300/60 p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/50">Available</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{updateStatus.availableVersion || 'Unknown'}</div>
        </div>
        <div className="surface-panel border border-base-300/60 p-4">
          <div className="text-xs uppercase tracking-[0.18em] text-base-content/50">Status</div>
          <div className="mt-2 text-lg font-semibold text-base-content">{updateStatus.updateAvailable ? 'Update available' : 'Up to date'}</div>
        </div>
      </div>

      <dl className="space-y-3">
        <Detail label="Installed version" value={updateStatus.installedVersion || 'Unknown'} />
        <Detail label="Available version" value={updateStatus.availableVersion || 'Unknown'} />
        <Detail label="Update available" value={updateStatus.updateAvailable ? 'Yes' : 'No'} />
      </dl>

      {confirmUpdate ? (
        <InlineNotice
          tone="warn"
          title="Confirm update"
          detail="A preventive backup will be created before updating Node-RED. Rollback will run automatically if health checks fail."
        />
      ) : null}

      <div className="flex gap-3 justify-end">
        {confirmUpdate ? (
          <>
            <button
              className="action-btn-ghost"
              type="button"
              onClick={() => setConfirmUpdate(false)}
              disabled={applyMutation.isPending}
            >
              Cancel
            </button>
            <button
              className="action-btn-primary"
              type="button"
              onClick={() => applyMutation.mutate()}
              disabled={busy || applyMutation.isPending}
            >
              {applyMutation.isPending ? 'Updating...' : 'Confirm update'}
            </button>
          </>
        ) : (
          <button
            className="action-btn-primary"
            type="button"
            onClick={() => setConfirmUpdate(true)}
            disabled={busy || !updateStatus.updateAvailable}
          >
            {updateStatus.updateAvailable ? 'Update Node-RED' : 'Up to date'}
          </button>
        )}
      </div>
    </div>
  )
}
