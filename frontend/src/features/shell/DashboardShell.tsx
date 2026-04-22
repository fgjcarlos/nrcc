import { useState } from 'react'
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
  const [drawerOpen, setDrawerOpen] = useState(false)
  const items: Array<{ to: string; label: string; page: PageKey }> = [
    { to: '/app/overview', label: 'Overview', page: 'overview' },
    { to: '/app/logs', label: 'Logs', page: 'logs' },
    { to: '/app/flows', label: 'Flows', page: 'flows' },
    { to: '/app/diagnostics', label: 'Diagnostics', page: 'diagnostics' },
    { to: '/app/config', label: 'Config', page: 'config' },
    { to: '/app/environment', label: 'Environment', page: 'environment' },
    { to: '/app/backups', label: 'Backups', page: 'backups' },
    { to: '/app/libraries', label: 'Libraries', page: 'libraries' },
    { to: '/app/updates', label: 'Updates', page: 'updates' },
  ]

  return (
    <div className="drawer bg-base-100 lg:drawer-open">
      <a className="skip-link" href="#main-content">
        Skip to main content
      </a>
      <input
        id="sidebar-drawer"
        type="checkbox"
        className="drawer-toggle"
        checked={drawerOpen}
        onChange={(event) => setDrawerOpen(event.target.checked)}
      />
      <div className="drawer-content flex min-h-screen flex-col app-shell">
        <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-primary via-primary/50 to-transparent pointer-events-none z-50"></div>
        
        <header className="navbar glass-panel sticky top-0 z-20 border-b px-3 sm:px-6 ghost-divider">
          <div className="navbar-start min-w-0 gap-2 sm:gap-3">
            <button
              type="button"
              aria-controls="primary-navigation"
              aria-expanded={drawerOpen}
              aria-label={drawerOpen ? 'Close navigation' : 'Open navigation'}
              className="btn btn-ghost btn-square lg:hidden"
              onClick={() => setDrawerOpen((current) => !current)}
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
            <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/15 text-primary shadow-glow">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.8" d="M13 3L4 14h7l-1 7 10-13h-7l0-5z" />
              </svg>
            </div>
            <div className="min-w-0 leading-tight">
              <span className="hidden text-xs uppercase tracking-[0.28em] text-base-content/60 sm:block">Command Center</span>
              <span className="block truncate font-semibold text-base-content">Node-RED Control Center</span>
            </div>
          </div>
          <div className="navbar-end gap-2 sm:gap-3">
            <span className="hidden rounded-full bg-base-300/60 px-3 py-1 text-xs font-medium text-base-content/70 xl:inline">
              {globalStatus.title}
            </span>
            <ThemeToggle />
            <button className="btn btn-ghost" onClick={onLogout} disabled={logoutBusy}>
              {logoutBusy ? 'Signing out…' : 'Sign out'}
            </button>
          </div>
        </header>

        <main id="main-content" tabIndex={-1} className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
          <div className="mx-auto w-full max-w-7xl">{children}</div>
        </main>
      </div>

      <div className="drawer-side z-30">
        <label htmlFor="sidebar-drawer" aria-label="Close navigation" className="drawer-overlay" />
        <aside className="glass-panel border-r ghost-divider flex min-h-full w-[min(20rem,calc(100vw-1.5rem))] flex-col justify-between">
          <div className="flex flex-col gap-6 p-6">
            <div className="border-b pb-6 ghost-divider">
              <div className="flex items-center gap-2 mb-3">
                <div className="h-2 w-2 rounded-full bg-primary opacity-70"></div>
                <p className="text-xs uppercase tracking-[0.26em] text-base-content/50 font-semibold">Orchestrator</p>
              </div>
              <h1 className="mt-2 text-2xl font-bold tracking-tight text-base-content">Node-RED</h1>
              <p className="text-sm text-base-content/70">Control Center</p>
            </div>

            <div
              className={`surface-panel border p-4 section-card ${
                globalStatus.tone === 'ok'
                  ? 'section-card--success'
                  : globalStatus.tone === 'warn'
                    ? 'section-card--warning'
                    : 'section-card--info'
              }`}
            >
              <div>
                <p className="text-xs uppercase tracking-[0.18em] text-base-content/55 font-semibold">System status</p>
                <p className="mt-2 text-base font-semibold text-base-content">{globalStatus.title}</p>
                <p className="mt-1 text-sm text-base-content/70">{globalStatus.detail}</p>
              </div>
            </div>

            <nav id="primary-navigation" aria-label="Primary">
              <ul className="menu gap-1 rounded-2xl bg-base-300/40 p-2">
                {items.map((item) => (
                  <li key={item.page}>
                    <NavLink
                      to={item.to}
                      className={({ isActive }) => (isActive ? 'menu-active' : '')}
                      onClick={() => setDrawerOpen(false)}
                    >
                      {item.label}
                    </NavLink>
                  </li>
                ))}
              </ul>
            </nav>
          </div>

          <div className="border-t p-6 ghost-divider">
            <div className="surface-card border p-4 section-card section-card--default">
              <p className="text-xs uppercase tracking-[0.18em] text-base-content/55 font-semibold">Active session</p>
              <p className="mt-3 font-semibold text-base-content">{user.username}</p>
              <p className="text-sm text-base-content/70">{user.role}</p>
            </div>
          </div>
        </aside>
      </div>
    </div>
  )
}
