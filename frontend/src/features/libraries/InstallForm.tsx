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
    <article className="surface-card mb-6 border border-base-300/60 p-6 md:p-7">
      <div className="flex items-start justify-between gap-4 mb-5">
        <div>
          <h3 className="text-xl font-semibold text-base-content">Install package</h3>
          <p className="mt-1 text-sm text-base-content/60">Add npm modules directly into the runtime environment.</p>
        </div>
      </div>
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
            className="action-btn-primary w-full"
            type="submit"
            disabled={busy || isPending || packageName.trim() === ''}
          >
            {isPending ? 'Installing...' : 'Install package'}
          </button>
        </form>
    </article>
  )
}
