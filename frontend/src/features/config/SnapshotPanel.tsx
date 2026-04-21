import { useEffect, useState } from 'react'
import { api, type OperationStatus } from '../../api'
import { ConfigSnapshot } from '../../types/config'
import { InlineNotice, LoadingState, EmptyState } from '../../common/components'

type SnapshotPanelProps = {
  isOpen: boolean
  operationStatus?: OperationStatus
  onClose: () => void
  onRestored: () => void
}

export function SnapshotPanel({ isOpen, operationStatus, onClose, onRestored }: SnapshotPanelProps) {
  const [snapshots, setSnapshots] = useState<ConfigSnapshot[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')
  const [creating, setCreating] = useState(false)
  const [newLabel, setNewLabel] = useState('')
  const [restoring, setRestoring] = useState<string | null>(null)
  const [confirmRestoreId, setConfirmRestoreId] = useState<string | null>(null)
  const busy = operationStatus?.busy ?? false

  // Fetch snapshots on open
  useEffect(() => {
    if (!isOpen) return

    const fetchSnapshots = async () => {
      setLoading(true)
      setError('')
      try {
        const result = await api.listConfigSnapshots()
        setSnapshots(result.items)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load snapshots')
      } finally {
        setLoading(false)
      }
    }

    fetchSnapshots()
  }, [isOpen])

  const handleCreate = async () => {
    if (creating) return
    setCreating(true)
    setError('')
    try {
      const snapshot = await api.createConfigSnapshot(newLabel || undefined)
      setSnapshots((prev) => [snapshot, ...prev])
      setNewLabel('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create snapshot')
    } finally {
      setCreating(false)
    }
  }

  const handleRestore = async (id: string) => {
    setRestoring(id)
    setError('')
    try {
      await api.restoreConfigSnapshot(id)
      onRestored()
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to restore snapshot')
    } finally {
      setRestoring(null)
      setConfirmRestoreId(null)
    }
  }

  if (!isOpen) {
    return null
  }

  return (
    <div className="modal-overlay">
      <div className="surface-card w-full max-w-2xl max-h-[32rem] overflow-auto border border-base-300/60 p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-base-content">Config Snapshots</h2>
          <button className="action-btn-ghost px-3 py-2" onClick={onClose} aria-label="Close snapshots">
            ✕
          </button>
        </div>

        <div className="space-y-4">
          {/* Create snapshot section */}
          <div className="surface-panel form-control mb-4 border border-base-300/60 p-4">
            <label className="form-field inline">
              <span className="label-text">New Snapshot Label (optional)</span>
              <input
                className="input input-bordered mt-2"
                type="text"
                value={newLabel}
                onChange={(e) => setNewLabel(e.target.value)}
                placeholder="e.g., Before upgrade..."
                disabled={creating}
              />
            </label>
            <button
              className="action-btn-primary mt-3"
              onClick={handleCreate}
              disabled={creating}
            >
              {creating ? 'Creating...' : 'Create Snapshot'}
            </button>
          </div>

          {error && (
            <p className="alert alert-error text-sm">
              <strong>Error:</strong> {error}
            </p>
          )}

          {busy && !restoring ? (
            <InlineNotice
              tone="warn"
              title="System busy"
              detail={
                (operationStatus?.type ? `${operationStatus.type} in progress` : 'Another operation is in progress') +
                (operationStatus?.detail ? `: ${operationStatus.detail}` : '.')
              }
            />
          ) : null}

          {/* Snapshots list */}
          {loading && <LoadingState message="Loading snapshots..." />}

          {!loading && snapshots.length === 0 && (
            <EmptyState
              title="No snapshots yet"
              description="Create one to get started."
            />
          )}

           {!loading && snapshots.length > 0 && (
             <div className="space-y-3">
               <table className="table w-full">
                  <thead>
                    <tr className="table-header-subtle">
                      <th>Created</th>
                      <th>Label</th>
                      <th>Reason</th>
                     <th>Action</th>
                   </tr>
                 </thead>
                 <tbody>
                   {snapshots.map((snapshot) => (
                     <tr key={snapshot.id} className="table-row-hover">
                       <td>{new Date(snapshot.createdAt).toLocaleString()}</td>
                       <td>{snapshot.label || '—'}</td>
                       <td>
                        <span className="badge">{snapshot.reason}</span>
                      </td>
                       <td>
                         {confirmRestoreId === snapshot.id ? (
                           <div className="flex gap-2 items-center">
                              <p className="text-sm">Restore this snapshot?</p>
                              <button
                                 className="action-btn-danger"
                                 onClick={() => handleRestore(snapshot.id)}
                                 disabled={busy || restoring === snapshot.id}
                               >
                               {restoring === snapshot.id ? 'Restoring...' : 'Confirm'}
                              </button>
                              <button
                                 className="action-btn-ghost"
                                onClick={() => setConfirmRestoreId(null)}
                                disabled={busy || restoring === snapshot.id}
                              >
                              Cancel
                            </button>
                          </div>
                         ) : (
                           <button
                             className="action-btn-secondary"
                             onClick={() => setConfirmRestoreId(snapshot.id)}
                              disabled={busy}
                            >
                             Restore
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <div className="flex gap-3 justify-end mt-6">
          <button className="action-btn-ghost" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
