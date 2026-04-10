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
        return 'border-warning'
      case 'neutral':
        return 'border-neutral'
      case 'ok':
      default:
        return 'border-success'
    }
  }

  return (
    <article className={`card bg-base-200 border-l-4 ${getAccentClass()} p-6 shadow`}>
      <p className="text-sm text-base-content opacity-75 uppercase tracking-wider">{label}</p>
      <h3 className="text-3xl font-bold text-base-content mt-2">{value}</h3>
    </article>
  )
}
