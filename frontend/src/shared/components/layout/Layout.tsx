import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';

export function Layout() {
  return (
    <div className="drawer lg:drawer-open min-h-screen bg-background app-shell">
      <input id="sidebar-drawer" type="checkbox" className="drawer-toggle" />

      <div className="drawer-content flex min-h-screen flex-col lg:pl-0">
        <Header />
        <main className="flex-1 overflow-y-auto px-3 py-4 sm:px-5 sm:py-6 lg:px-6">
          <section
            data-testid="page-content-shell"
            className="surface-panel min-h-[calc(100vh-7rem)] rounded-2xl border p-4 sm:p-6"
          >
            <Outlet />
          </section>
        </main>
      </div>

      <div className="drawer-side z-50">
        <label htmlFor="sidebar-drawer" aria-label="close sidebar" className="drawer-overlay"></label>
        <Sidebar />
      </div>
    </div>
  );
}
