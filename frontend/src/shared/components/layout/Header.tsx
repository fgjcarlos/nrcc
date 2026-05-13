import { Menu, RadioTower } from 'lucide-react';
import { ThemeToggle } from '@/shared/components';
import { UpdateNotificationChip } from '@/features/updates/components/UpdateNotificationChip';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001';

export function Header() {

  return (
    <>
      <div className="px-4 border-b navbar glass-panel ghost-divider sm:px-6">
        <div className="navbar-start">
          <label htmlFor="sidebar-drawer" className="mr-2 rounded-lg btn btn-ghost btn-square lg:hidden">
            <Menu className="w-5 h-5" />
          </label>
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-10 h-10 border rounded-lg border-accent/20 bg-accent/10 text-accent">
              <RadioTower className="h-5 w-5 stroke-[1.8]" />
            </div>
            <div className="leading-tight">
              <span className="block text-xs uppercase tracking-[0.22em] text-base-content/55">Command Center</span>
              <span className="block font-semibold text-base-content">Node-RED Control Center</span>
            </div>
          </div>
        </div>

        <div className="gap-2 navbar-end sm:gap-3">
          <UpdateNotificationChip />
          <ThemeToggle />
          <span className="hidden px-3 py-1 text-xs font-medium border rounded-lg border-border bg-base-300/40 text-base-content/70 xl:inline">
            API: {API_URL}
          </span>
        </div>
      </div>
    </>
  );
}
