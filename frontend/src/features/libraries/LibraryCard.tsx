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
    <article className="library-card" key={item.name}>
      <div className="library-card-copy">
        <strong>{item.name}</strong>
        <p>{item.version || 'Unknown version'}</p>
      </div>
      <button
        className="ghost-button"
        type="button"
        onClick={onUninstall}
        disabled={busy || isPending}
      >
        {isPending ? 'Working...' : 'Remove'}
      </button>
    </article>
  )
}
