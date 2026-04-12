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
    <article className="card bg-base-200 shadow mb-6">
      <div className="card-body">
        <h3 className="card-title text-2xl">Install package</h3>
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            onSubmit()
          }}
        >
          <div className="form-control">
            <label className="label">
              <span className="label-text font-medium">Package name</span>
            </label>
            <input
              className="input input-bordered bg-base-100"
              value={packageName}
              onChange={(event) => onChange(event.target.value)}
              placeholder="@scope/package"
            />
          </div>
          <button
            className="btn btn-primary w-full"
            type="submit"
            disabled={busy || isPending || packageName.trim() === ''}
          >
            {isPending ? 'Installing...' : 'Install package'}
          </button>
        </form>
      </div>
    </article>
  )
}
