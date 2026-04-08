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
      <>
        <button
          className="ghost-button"
          type="button"
          onClick={onCancel}
          disabled={restarting}
        >
          Cancel
        </button>
        <button
          className="primary-button"
          type="button"
          onClick={onConfirm}
          disabled={restarting}
        >
          {restarting ? 'Restarting...' : 'Confirm restart'}
        </button>
      </>
    )
  }

  return (
    <button className="primary-button" type="button" onClick={onRequest} disabled={restarting}>
      Restart Node-RED
    </button>
  )
}
