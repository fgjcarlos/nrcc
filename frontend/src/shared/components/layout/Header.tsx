import { Menu, RadioTower } from 'lucide-react';
import { ThemeToggle } from '@/shared/components';
import { UpdateNotificationChip } from '@/features/updates/components/UpdateNotificationChip';
import { CommandPalette } from '@/shared/components/command-palette';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001';

export function Header() {

  return (
    <header
      data-testid="app-topbar"
      className="app-topbar-shell sticky top-0 z-40 border-b px-3 py-3 sm:px-5"
    >
      <div className="flex min-h-14 items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <label
            htmlFor="sidebar-drawer"
            className="btn btn-ghost btn-square btn-sm rounded-xl border border-border/70 bg-base-300/45 text-base-content/80 lg:hidden"
            aria-label="Abrir navegación principal"
          >
            <Menu className="w-5 h-5" />
          </label>
          <div className="flex min-w-0 items-center gap-3">
            <div className="topbar-signal-icon flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border text-accent">
              <RadioTower className="h-5 w-5 stroke-[1.8]" />
            </div>
            <div className="min-w-0 leading-tight">
              <span className="block text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-base-content/55">Command Center</span>
              <span className="block truncate text-sm font-semibold text-base-content sm:text-base">Node-RED Control Center</span>
            </div>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2 sm:gap-3">
          <CommandPalette />
          <UpdateNotificationChip />
          <ThemeToggle />
          <span className="api-status-chip hidden max-w-[18rem] truncate rounded-xl border px-3 py-2 text-xs font-medium text-base-content/70 xl:inline-flex">
            API: {API_URL}
          </span>
        </div>
      </div>
    </header>
  );
}
