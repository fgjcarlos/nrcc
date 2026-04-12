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
    <article className={`card stat-card bg-base-200 p-6 shadow-elevation-2 rounded-lg ${getAccentClass()}`}>
      <p className="text-sm text-base-content opacity-75 uppercase tracking-wider stat-card-label">{label}</p>
      <h3 className="text-3xl font-bold text-base-content mt-2">{value}</h3>
    </article>
  )
}
