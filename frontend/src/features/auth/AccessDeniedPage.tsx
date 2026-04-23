import { PageTransition, InlineNotice } from '../../common/components'

export function AccessDeniedPage({
  userRole,
  onLogout,
  logoutBusy,
}: {
  userRole: string
  onLogout: () => void
  logoutBusy: boolean
}) {
  return (
    <PageTransition>
      <main id="auth-main" tabIndex={-1} className="min-h-screen bg-base-100 px-6 py-16">
        <div className="mx-auto flex max-w-2xl flex-col gap-6">
          <header>
            <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">Access denied</p>
            <h1 className="mt-2 text-3xl font-semibold text-base-content">This first slice is admin-only</h1>
            <p className="mt-3 text-sm text-base-content/70">
              Signed in as <span className="font-semibold">{userRole}</span>. Viewer and operator rollout is intentionally deferred, so this account cannot use the control center yet.
            </p>
          </header>
          <InlineNotice
            tone="info"
            title="Administrator required"
            detail="Ask an administrator to promote this account or sign in with an existing administrator account."
          />
          <div>
            <button className="btn btn-primary" type="button" onClick={onLogout} disabled={logoutBusy}>
              {logoutBusy ? 'Signing out...' : 'Sign out'}
            </button>
          </div>
        </div>
      </main>
    </PageTransition>
  )
}
