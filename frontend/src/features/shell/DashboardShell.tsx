import { NavLink } from 'react-router-dom'
import type { User } from '../../api'
import type { PageKey, GlobalStatus } from '../../common/types'

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
    <main className="app-shell">
      <aside className="sidebar">
        <div className="sidebar-top">
          <div>
            <p className="eyebrow">NRCC</p>
            <h1>Control Center</h1>
            <p className="sidebar-copy">
              Local-first operations for Node-RED with Go runtime control and cookie-backed sessions.
            </p>
          </div>

          <section className={`status-banner ${globalStatus.tone}`}>
            <div className="status-banner-copy">
              <p className="status-banner-label">System status</p>
              <strong>{globalStatus.title}</strong>
              <p>{globalStatus.detail}</p>
            </div>
          </section>

          <nav className="sidebar-nav" aria-label="Primary">
            {items.map((item) => (
              <NavLink
                key={item.page}
                to={item.to}
                className={({ isActive }) =>
                  isActive ? 'nav-link active' : 'nav-link'
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>
        </div>

        <div className="profile-card">
          <p className="profile-name">{user.username}</p>
          <p className="profile-role">{user.role}</p>
          <button className="ghost-button wide" type="button" onClick={onLogout} disabled={logoutBusy}>
            {logoutBusy ? 'Signing out...' : 'Sign out'}
          </button>
        </div>
      </aside>

      <section className="content">{children}</section>
    </main>
  )
}
