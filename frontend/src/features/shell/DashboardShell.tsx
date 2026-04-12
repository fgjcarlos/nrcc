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
    <div className="drawer lg:drawer-open">
      <input id="sidebar-drawer" type="checkbox" className="drawer-toggle" />
      <div className="drawer-content flex flex-col">
        {/* Sticky Navbar Header */}
        <header className="navbar bg-base-200 sticky top-0 z-10 shadow-elevation-2 rounded-lg">
          <div className="flex-1">
            <label htmlFor="sidebar-drawer" className="btn btn-ghost btn-circle lg:hidden">
              <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </label>
            <span className="text-lg font-semibold">Node-RED Control Center</span>
          </div>
          <div className="navbar-end gap-2">
            <ThemeToggle />
            <button className="btn btn-ghost btn-sm" onClick={onLogout} disabled={logoutBusy}>
              {logoutBusy ? 'Signing out…' : 'Sign out'}
            </button>
          </div>
        </header>

        {/* Main Content Area */}
        <main className="flex-1 p-6 bg-base-100">
          {children}
        </main>
      </div>

      {/* Sidebar (Drawer Side) */}
      <div className="drawer-side">
        <label htmlFor="sidebar-drawer" className="drawer-overlay" />
        <aside className="bg-base-200 w-72 min-h-full flex flex-col justify-between p-6">
          {/* Sidebar Top */}
          <div className="flex flex-col gap-6">
            <div>
              <p className="text-xs font-semibold text-base-content opacity-60 uppercase tracking-wide">NRCC</p>
              <h1 className="text-2xl font-bold text-base-content">Control Center</h1>
              <p className="text-sm text-base-content opacity-70 mt-2">
                Local-first operations for Node-RED with Go runtime control and cookie-backed sessions.
              </p>
            </div>

            {/* Status Banner */}
            <div className={`flex gap-3 p-4 rounded-lg bg-base-200 border border-[color:var(--border-neutral)] ${
              globalStatus.tone === 'ok' ? 'border-l-success' : 
              globalStatus.tone === 'warn' ? 'border-l-warning' : 
              'border-l-info'
            } border-l-4 shadow-elevation-1`}>
              <div>
                <p className="text-xs font-semibold opacity-75">System status</p>
                <p className="font-bold text-base">{globalStatus.title}</p>
                <p className="text-sm opacity-75">{globalStatus.detail}</p>
              </div>
            </div>

            {/* Navigation Menu */}
            <nav className="menu bg-base-300 rounded-lg p-2" aria-label="Primary">
              {items.map((item) => (
                <li key={item.page}>
                  <NavLink
                    to={item.to}
                    className={({ isActive }) => (isActive ? 'active' : '')}
                  >
                    {item.label}
                  </NavLink>
                </li>
              ))}
            </nav>
          </div>

          {/* Profile Card (Sidebar Bottom) */}
          <div className="card bg-base-300 shadow-elevation-2 p-4 rounded-lg">
            <div className="card-body p-0">
              <p className="font-bold text-base-content">{user.username}</p>
              <p className="text-sm text-base-content opacity-70">{user.role}</p>
            </div>
          </div>
        </aside>
      </div>
    </div>
  )
}
