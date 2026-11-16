import { useRef, useState } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { SkipLink } from '@/shared/components/a11y/SkipLink';
import { useBelowLg } from '@/shared/hooks/useBelowLg';
import { useModalKeyboard } from '@/shared/hooks/useModalKeyboard';
import { useT } from '@/i18n';

/** Mobile navigation drawer: a modal with focus trap, Escape and focus return. */
function NavDrawer({ onClose }: { onClose: () => void }) {
  const { t } = useT();
  const panelRef = useRef<HTMLDivElement>(null);
  useModalKeyboard({ isOpen: true, containerRef: panelRef, onEscape: onClose });

  return (
    <div className="fixed inset-0 z-50">
      <div
        className="motion-scrim absolute inset-0 bg-[color-mix(in_srgb,var(--ds-rail-bg)_55%,transparent)]"
        aria-hidden="true"
        onClick={onClose}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label={t('common:layout.mainNavigation')}
        tabIndex={-1}
        className="motion-overlay relative h-full w-64 max-w-[85vw] [--overlay-from:-8px_0]"
      >
        <Sidebar mode="drawer" onClose={onClose} />
      </div>
    </div>
  );
}

/**
 * Layout: the shell shared by every authenticated view. The ink rail
 * sits beside the content from `lg` up; below it the header's menu
 * button opens the rail as a drawer. The SkipLink is the first focusable
 * element and jumps to the main landmark (tabindex=-1).
 */
export function Layout() {
  const { t } = useT();
  const belowLg = useBelowLg();
  const { pathname } = useLocation();
  // The drawer belongs to the route it was opened on, so following a link
  // (or any navigation) closes it without an effect.
  const [drawerPath, setDrawerPath] = useState<string | null>(null);
  const drawerOpen = belowLg && drawerPath === pathname;

  return (
    <div className="app-shell flex min-h-screen">
      <SkipLink targetId="main-content" label={t('common:layout.skipToContent')} />

      {!belowLg && (
        <div className="sticky top-0 h-screen shrink-0">
          <Sidebar />
        </div>
      )}

      <div className="flex min-h-screen min-w-0 flex-1 flex-col">
        <Header
          compact={belowLg}
          navOpen={drawerOpen}
          onOpenNav={() => setDrawerPath(pathname)}
        />
        <main
          id="main-content"
          tabIndex={-1}
          className="flex-1 px-3 py-4 sm:px-5 sm:py-6 lg:px-6 focus:outline-none"
        >
          <section
            data-testid="page-content-shell"
            className="surface-panel min-h-[calc(100vh-7rem)] rounded-box border p-4 sm:p-6"
          >
            <Outlet />
          </section>
        </main>
      </div>

      {drawerOpen && <NavDrawer onClose={() => setDrawerPath(null)} />}
    </div>
  );
}
