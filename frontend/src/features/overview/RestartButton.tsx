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
      <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
        <button
          className="btn btn-ghost w-full sm:w-auto"
          type="button"
          onClick={onCancel}
          disabled={restarting}
        >
          Cancel
        </button>
        <button
          className="btn btn-primary w-full sm:w-auto"
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
    <button className="btn btn-primary w-full sm:w-auto" type="button" onClick={onRequest} disabled={restarting}>
      Restart Node-RED
    </button>
  )
}
