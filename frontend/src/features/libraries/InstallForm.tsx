import { FormField } from '../../components/forms'

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
    <article className="card bg-base-200 mb-6">
      <div className="card-body">
        <h3 className="card-title text-2xl">Install package</h3>
        <form
          className="space-y-4"
          onSubmit={(event) => {
            event.preventDefault()
            onSubmit()
          }}
        >
          <FormField
            id="install-form-package-name"
            label="Package name"
            type="text"
            placeholder="@scope/package"
            value={packageName}
            onChange={onChange}
            disabled={busy || isPending}
          />
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
