export function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="detail-row grid grid-cols-1 gap-2 py-3 sm:grid-cols-[160px_minmax(0,1fr)] sm:gap-4">
      <dt className="text-xs font-medium uppercase tracking-[0.16em] text-base-content/50">{label}</dt>
      <dd className="text-sm text-base-content sm:text-right">{value}</dd>
    </div>
  )
}
