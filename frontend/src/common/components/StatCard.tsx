export function StatCard({
  label,
  value,
  accent,
}: {
  label: string
  value: string
  accent: 'ok' | 'warn' | 'neutral'
}) {
  const getAccentClass = () => {
    switch (accent) {
      case 'warn':
        return 'stat-card--warn'
      case 'neutral':
        return 'stat-card--neutral'
      case 'ok':
      default:
        return 'stat-card--ok'
    }
  }

  return (
    <article className={`stat-card p-6 ${getAccentClass()}`}>
      <p className="stat-card-label text-xs uppercase tracking-[0.22em] text-base-content/55">{label}</p>
      <h3 className="mt-3 text-3xl font-bold tracking-tight text-base-content">{value}</h3>
    </article>
  )
}
