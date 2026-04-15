export function LoadingScreen({ label }: { label: string }) {
  return (
    <main className="flex flex-col items-center justify-center min-h-screen bg-base-100">
      <section className="card bg-base-200 w-full max-w-md p-8">
        <p className="text-xs font-semibold text-primary uppercase tracking-wide">NRCC</p>
        <h1 className="text-2xl font-bold mt-4 text-base-content">{label}</h1>
      </section>
    </main>
  )
}
