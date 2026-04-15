import { useEffect, useState } from 'react'
import { api } from '../../api'
import { ConfigSnapshot } from '../../types/config'

type SnapshotPanelProps = {
  isOpen: boolean
  onClose: () => void
  onRestored: () => void
}

export function SnapshotPanel({ isOpen, onClose, onRestored }: SnapshotPanelProps) {
  const [snapshots, setSnapshots] = useState<ConfigSnapshot[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')
  const [creating, setCreating] = useState(false)
  const [newLabel, setNewLabel] = useState('')
  const [restoring, setRestoring] = useState<string | null>(null)
  const [confirmRestoreId, setConfirmRestoreId] = useState<string | null>(null)

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
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
      <div className="card bg-base-200 w-full max-w-2xl max-h-96 overflow-auto">
        <div className="flex items-center justify-between mb-4">
          <h2>Config Snapshots (Backups)</h2>
          <button className="btn btn-ghost btn-sm btn-circle" onClick={onClose} aria-label="Close snapshots">
            ✕
          </button>
        </div>

        <div className="space-y-4">
          {/* Create snapshot section */}
          <div className="form-control mb-4">
            <label className="form-field inline">
              <span>New Snapshot Label (optional)</span>
              <input
                type="text"
                value={newLabel}
                onChange={(e) => setNewLabel(e.target.value)}
                placeholder="e.g., Before upgrade..."
                disabled={creating}
              />
            </label>
            <button
              className="btn btn-primary"
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

          {/* Snapshots list */}
          {loading && <p className="text-sm text-base-content/60">Loading snapshots...</p>}

          {!loading && snapshots.length === 0 && (
            <p className="text-sm text-base-content/60">No snapshots yet. Create one to get started.</p>
          )}

           {!loading && snapshots.length > 0 && (
             <div className="space-y-3">
               <table className="table table-zebra w-full">
                 <thead>
                   <tr>
                     <th>Created</th>
                     <th>Label</th>
                     <th>Reason</th>
                     <th>Action</th>
                   </tr>
                 </thead>
                <tbody>
                  {snapshots.map((snapshot) => (
                    <tr key={snapshot.id}>
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
                               className="btn btn-error btn-sm"
                               onClick={() => handleRestore(snapshot.id)}
                               disabled={restoring === snapshot.id}
                             >
                               {restoring === snapshot.id ? 'Restoring...' : 'Confirm'}
                             </button>
                             <button
                               className="btn btn-ghost btn-sm"
                              onClick={() => setConfirmRestoreId(null)}
                              disabled={restoring === snapshot.id}
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <button
                            className="btn btn-ghost"
                            onClick={() => setConfirmRestoreId(snapshot.id)}
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
          <button className="btn btn-ghost" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
