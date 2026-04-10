export function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-2 gap-4 py-3 border-b border-base-300">
      <dt className="text-sm font-semibold text-base-content opacity-75">{label}</dt>
      <dd className="text-sm text-base-content">{value}</dd>
    </div>
  )
}
