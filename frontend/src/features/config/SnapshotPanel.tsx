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
    <div className="snapshot-panel modal-overlay">
      <div className="modal-content">
        <div className="modal-header">
          <h2>Config Snapshots (Backups)</h2>
          <button className="close-button" onClick={onClose} aria-label="Close snapshots">
            ✕
          </button>
        </div>

        <div className="modal-body">
          {/* Create snapshot section */}
          <div className="snapshot-create">
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
              className="primary-button"
              onClick={handleCreate}
              disabled={creating}
            >
              {creating ? 'Creating...' : 'Create Snapshot'}
            </button>
          </div>

          {error && (
            <p className="field-error">
              <strong>Error:</strong> {error}
            </p>
          )}

          {/* Snapshots list */}
          {loading && <p className="muted">Loading snapshots...</p>}

          {!loading && snapshots.length === 0 && (
            <p className="muted">No snapshots yet. Create one to get started.</p>
          )}

          {!loading && snapshots.length > 0 && (
            <div className="snapshot-list">
              <table>
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
                          <div className="confirm-actions">
                            <p className="confirm-text">Restore this snapshot?</p>
                            <button
                              className="danger-button"
                              onClick={() => handleRestore(snapshot.id)}
                              disabled={restoring === snapshot.id}
                            >
                              {restoring === snapshot.id ? 'Restoring...' : 'Confirm'}
                            </button>
                            <button
                              className="secondary-button"
                              onClick={() => setConfirmRestoreId(null)}
                              disabled={restoring === snapshot.id}
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <button
                            className="secondary-button"
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

        <div className="modal-footer">
          <button className="secondary-button" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
