export function RestartButton({
  confirmRestart,
  blocked,
  restarting,
  onConfirm,
  onCancel,
  onRequest,
}: {
  confirmRestart: boolean
  blocked: boolean
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
          disabled={blocked || restarting}
        >
          Cancel
        </button>
        <button
          className="btn btn-primary w-full sm:w-auto"
          type="button"
          onClick={onConfirm}
          disabled={blocked || restarting}
        >
          {restarting ? 'Restarting...' : blocked ? 'Restart blocked' : 'Confirm restart'}
        </button>
      </div>
    )
  }

  return (
    <button className="btn btn-primary w-full sm:w-auto" type="button" onClick={onRequest} disabled={blocked || restarting}>
      {restarting ? 'Restarting...' : blocked ? 'Restart blocked' : 'Restart Node-RED'}
    </button>
  )
}
