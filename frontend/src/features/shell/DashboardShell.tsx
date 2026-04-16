import { NavLink } from 'react-router-dom'
import type { User } from '../../api'
import type { PageKey, GlobalStatus } from '../../common/types'
import { ThemeToggle } from '../../common/components'

export function DashboardShell({
  user,
  globalStatus,
  logoutBusy,
  onLogout,
  children,
}: {
  user: User
  globalStatus: GlobalStatus
  logoutBusy: boolean
  onLogout: () => void
  children: React.ReactNode
}) {
  const items: Array<{ to: string; label: string; page: PageKey }> = [
    { to: '/app/overview', label: 'Overview', page: 'overview' },
    { to: '/app/logs', label: 'Logs', page: 'logs' },
    { to: '/app/diagnostics', label: 'Diagnostics', page: 'diagnostics' },
    { to: '/app/config', label: 'Config', page: 'config' },
    { to: '/app/environment', label: 'Environment', page: 'environment' },
    { to: '/app/backups', label: 'Backups', page: 'backups' },
    { to: '/app/libraries', label: 'Libraries', page: 'libraries' },
    { to: '/app/updates', label: 'Updates', page: 'updates' },
  ]

  return (
    <div className="drawer lg:drawer-open bg-base-100">
      <input id="sidebar-drawer" type="checkbox" className="drawer-toggle" />
      <div className="drawer-content flex min-h-screen flex-col app-shell">
        <header className="navbar glass-panel sticky top-0 z-20 border-b px-4 sm:px-6 ghost-divider">
          <div className="navbar-start gap-3">
            <label htmlFor="sidebar-drawer" className="btn btn-ghost btn-square lg:hidden">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </label>
            <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/15 text-primary shadow-glow">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.8" d="M13 3L4 14h7l-1 7 10-13h-7l0-5z" />
              </svg>
            </div>
            <div className="leading-tight">
              <span className="block text-xs uppercase tracking-[0.28em] text-base-content/60">Command Center</span>
              <span className="block font-semibold text-base-content">Node-RED Control Center</span>
            </div>
          </div>
          <div className="navbar-end gap-3">
            <span className="hidden rounded-full bg-base-300/60 px-3 py-1 text-xs font-medium text-base-content/70 xl:inline">
              {globalStatus.title}
            </span>
            <ThemeToggle />
            <button className="btn btn-ghost btn-sm" onClick={onLogout} disabled={logoutBusy}>
              {logoutBusy ? 'Signing out…' : 'Sign out'}
            </button>
          </div>
        </header>

        <main className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-7xl">{children}</div>
        </main>
      </div>

      <div className="drawer-side z-30">
        <label htmlFor="sidebar-drawer" className="drawer-overlay" />
        <aside className="glass-panel border-r ghost-divider flex min-h-full w-72 flex-col justify-between">
          <div className="flex flex-col gap-6 p-6">
            <div className="border-b pb-6 ghost-divider">
              <p className="text-xs uppercase tracking-[0.26em] text-base-content/50">Orchestrator</p>
              <h1 className="mt-2 text-2xl font-bold tracking-tight text-base-content">Node-RED</h1>
              <p className="text-sm text-base-content/70">Control Center</p>
            </div>

            <div
              className={`surface-panel border p-4 ${
                globalStatus.tone === 'ok'
                  ? 'border-success/20'
                  : globalStatus.tone === 'warn'
                    ? 'border-warning/20'
                    : 'border-info/20'
              }`}
            >
              <div>
                <p className="text-xs uppercase tracking-[0.18em] text-base-content/55">System status</p>
                <p className="mt-2 text-base font-semibold text-base-content">{globalStatus.title}</p>
                <p className="mt-1 text-sm text-base-content/70">{globalStatus.detail}</p>
              </div>
            </div>

            <nav aria-label="Primary">
              <ul className="menu gap-1 rounded-2xl bg-base-300/40 p-2">
                {items.map((item) => (
                  <li key={item.page}>
                    <NavLink to={item.to} className={({ isActive }) => (isActive ? 'menu-active' : '')}>
                      {item.label}
                    </NavLink>
                  </li>
                ))}
              </ul>
            </nav>
          </div>

          <div className="border-t p-6 ghost-divider">
            <div className="surface-card border p-4">
              <p className="text-xs uppercase tracking-[0.18em] text-base-content/55">Active session</p>
              <p className="mt-3 font-semibold text-base-content">{user.username}</p>
              <p className="text-sm text-base-content/70">{user.role}</p>
            </div>
          </div>
        </aside>
      </div>
    </div>
  )
}
