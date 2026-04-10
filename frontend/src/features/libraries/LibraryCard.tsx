import type { LibraryPackage } from '../../api'

export function LibraryCard({
  item,
  isPending,
  busy,
  onUninstall,
}: {
  item: LibraryPackage
  isPending: boolean
  busy: boolean
  onUninstall: () => void
}) {
  return (
    <article className="card bg-base-200 shadow p-6 flex flex-row items-center justify-between" key={item.name}>
      <div className="flex-1">
        <strong className="text-base text-base-content">{item.name}</strong>
        <p className="text-sm text-base-content opacity-75 mt-1">{item.version || 'Unknown version'}</p>
      </div>
      <button
        className="btn btn-ghost btn-sm ml-4"
        type="button"
        onClick={onUninstall}
        disabled={busy || isPending}
      >
        {isPending ? 'Working...' : 'Remove'}
      </button>
    </article>
  )
}
