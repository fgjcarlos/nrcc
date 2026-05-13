export function DashboardHeader() {
  return (
    <div className="flex items-end justify-between gap-4">
      <div>
        <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">System overview</p>
        <h1 className="text-3xl font-bold tracking-tight text-base-content">Dashboard</h1>
      </div>
      <div className="hidden rounded-full bg-base-300/60 px-4 py-2 text-xs font-medium text-base-content/70 md:block">
        Live telemetry
      </div>
    </div>
  );
}
