export function LoadingScreen({ label }: { label: string }) {
  return (
    <main className="auth-shell">
      <section className="auth-panel loading-panel">
        <p className="eyebrow">NRCC</p>
        <h1>{label}</h1>
      </section>
    </main>
  )
}
