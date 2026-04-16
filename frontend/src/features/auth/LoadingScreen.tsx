export function LoadingScreen({ label }: { label: string }) {
  return (
    <main className="auth-shell flex min-h-screen items-center justify-center px-6 py-12">
      <section className="surface-card w-full max-w-md border border-base-300 p-8">
        <p className="text-xs uppercase tracking-[0.32em] text-primary/80">NRCC</p>
        <h1 className="mt-4 text-2xl font-bold tracking-tight text-base-content">{label}</h1>
      </section>
    </main>
  )
}
