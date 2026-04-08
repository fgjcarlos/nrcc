export function StatCard({
  label,
  value,
  accent,
}: {
  label: string
  value: string
  accent: 'ok' | 'warn' | 'neutral'
}) {
  return (
    <article className={`stat-card ${accent}`}>
      <p className="stat-label">{label}</p>
      <h3>{value}</h3>
    </article>
  )
}
