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
    <article className="surface-panel flex flex-col gap-4 border border-base-300/60 p-5 md:flex-row md:items-center md:justify-between" key={item.name}>
      <div className="flex-1 min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <strong className="text-base text-base-content">{item.name}</strong>
          <span className="rounded-full bg-base-300/60 px-2.5 py-1 text-xs text-base-content/70">
            {item.version || 'Unknown version'}
          </span>
        </div>
        <p className="text-sm text-base-content/65 mt-2">Installed runtime package managed by the control center.</p>
      </div>
      <button
        className="action-btn-danger md:ml-4"
        type="button"
        onClick={onUninstall}
        disabled={busy || isPending}
      >
        {isPending ? 'Working...' : 'Remove'}
      </button>
    </article>
  )
}
