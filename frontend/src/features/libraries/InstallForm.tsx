export function InstallForm({
  packageName,
  busy,
  isPending,
  onChange,
  onSubmit,
}: {
  packageName: string
  busy: boolean
  isPending: boolean
  onChange: (value: string) => void
  onSubmit: () => void
}) {
  return (
    <article className="panel">
      <div className="panel-header">
        <h3>Install package</h3>
      </div>
      <form
        className="library-form"
        onSubmit={(event) => {
          event.preventDefault()
          onSubmit()
        }}
      >
        <label>
          <span>Package name</span>
          <input
            value={packageName}
            onChange={(event) => onChange(event.target.value)}
            placeholder="@scope/package"
          />
        </label>
        <button
          className="primary-button"
          type="submit"
          disabled={busy || isPending || packageName.trim() === ''}
        >
          {isPending ? 'Installing...' : 'Install package'}
        </button>
      </form>
    </article>
  )
}
