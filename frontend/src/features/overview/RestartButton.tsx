export function RestartButton({
  confirmRestart,
  restarting,
  onConfirm,
  onCancel,
  onRequest,
}: {
  confirmRestart: boolean
  restarting: boolean
  onConfirm: () => void
  onCancel: () => void
  onRequest: () => void
}) {
  if (confirmRestart) {
    return (
      <div className="flex gap-2">
        <button
          className="btn btn-ghost btn-sm"
          type="button"
          onClick={onCancel}
          disabled={restarting}
        >
          Cancel
        </button>
        <button
          className="btn btn-primary btn-sm"
          type="button"
          onClick={onConfirm}
          disabled={restarting}
        >
          {restarting ? 'Restarting...' : 'Confirm restart'}
        </button>
      </div>
    )
  }

  return (
    <button className="btn btn-primary btn-sm" type="button" onClick={onRequest} disabled={restarting}>
      Restart Node-RED
    </button>
  )
}
