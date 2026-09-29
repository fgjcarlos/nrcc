import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { SkipLink } from '@/shared/components/a11y/SkipLink';
import { useT } from '@/i18n';

/**
 * Layout — issue #766 slices B/C/G
 *
 * Persistent shell shared by every authenticated view:
 *  - Sidebar (slice C) on lg+, collapses to a drawer on smaller
 *    viewports.
 *  - Header (slice C) carrying the runtime-context chips and
 *    application chrome (command palette, locale, theme).
 *  - A single <main id="main-content" tabindex={-1}> landmark where
 *    views are rendered via <Outlet />.
 *
 * Issue #766 slice G W2 adds the SkipLink as the first focusable
 * element so the first Tab press exposes it, then jumps the keyboard
 * user straight to the main content. `<main tabindex={-1}>` makes the
 * jump target focusable so the browser scrolls and moves focus to the
 * landmark rather than only scrolling.
 */
export function Layout() {
  const { t } = useT();
  return (
    <div className="drawer lg:drawer-open min-h-screen bg-background app-shell">
      <SkipLink targetId="main-content" label={t('common:layout.skipToContent')} />

      <input id="sidebar-drawer" type="checkbox" className="drawer-toggle" />

      <div className="drawer-content flex min-h-screen flex-col lg:pl-0">
        <Header />
        <main
          id="main-content"
          tabIndex={-1}
          className="flex-1 overflow-y-auto px-3 py-4 sm:px-5 sm:py-6 lg:px-6 focus:outline-none"
        >
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