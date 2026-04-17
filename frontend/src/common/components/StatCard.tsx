export function StatCard({
  label,
  value,
  accent,
  detail,
}: {
  label: string
  value: string
  accent: 'ok' | 'warn' | 'neutral' | 'error' | 'info'
  detail?: string
}) {
  const getAccentClass = () => {
    switch (accent) {
      case 'warn':
        return 'stat-card--warn'
      case 'error':
        return 'stat-card--error'
      case 'info':
        return 'stat-card--info'
      case 'neutral':
        return 'stat-card--neutral'
      case 'ok':
      default:
        return 'stat-card--ok'
    }
  }

  const getIconColor = () => {
    switch (accent) {
      case 'warn':
        return 'text-warning'
      case 'error':
        return 'text-error'
      case 'info':
        return 'text-info'
      case 'ok':
        return 'text-success'
      default:
        return 'text-base-content/60'
    }
  }

  return (
    <article className={`stat-card surface-card relative overflow-hidden border p-6 ${getAccentClass()}`}>
      <p className="stat-card-label text-xs uppercase tracking-[0.22em] text-base-content/55 font-semibold">
        {label}
      </p>
      <h3 className="mt-3 text-3xl font-bold tracking-tight text-base-content">{value}</h3>
      {detail ? <p className="mt-2 text-sm text-base-content/65">{detail}</p> : null}
      <div className={`mt-3 h-0.5 w-8 rounded-full bg-current opacity-30 ${getIconColor()}`} />
    </article>
  )
}
