import { useState, useEffect } from 'react';
import { NavLink } from 'react-router-dom';
import {
  Blocks,
  Box,
  ChevronLeft,
  ChevronRight,
  Container,
  DatabaseBackup,
  File,
  FileTerminal,
  Gauge,
  HardDrive,
  Library,
  RotateCw,
  Settings,
  SlidersHorizontal,
  UsersRound,
  Workflow
} from 'lucide-react';
import { cn } from '@/shared/lib/utils';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { UserMenu } from './UserMenu';

const navItems = [
  { to: '/dashboard', label: 'Dashboard', icon: Gauge },
  { to: '/bootstrap', label: 'Bootstrap', icon: HardDrive },
  { to: '/configuration', label: 'Configuration', icon: Settings },
  { to: '/logs', label: 'Logs', icon: FileTerminal },
  { to: '/docker', label: 'Docker', icon: Container },
  { to: '/flows', label: 'Flows', icon: Workflow },
  { to: '/backups', label: 'Backups', icon: DatabaseBackup },
  { to: '/files', label: 'Files', icon: File },
  { to: '/updates', label: 'Updates', icon: RotateCw },
  { to: '/libraries', label: 'Libraries', icon: Library },
  { to: '/environment', label: 'Environment', icon: SlidersHorizontal },
];

const adminItems = [
  { to: '/settings/users', label: 'Users', icon: UsersRound },
];

const STORAGE_KEY = 'sidebar-collapsed';
const MOBILE_BREAKPOINT = 768;

export function Sidebar() {
  const { user, logout } = useAuth();
  const isAdmin = user?.role === 'admin';
  
  const [isMobile, setIsMobile] = useState(false);
  const [collapsed, setCollapsed] = useState(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored ? stored === 'true' : false;
  });

  useEffect(() => {
    const checkMobile = () => {
      setIsMobile(window.innerWidth < MOBILE_BREAKPOINT);
    };
    
    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, String(collapsed));
  }, [collapsed]);

  const toggleCollapsed = () => setCollapsed(!collapsed);

  const handleLogout = () => {
    logout();
    window.location.href = '/login';
  };

  const navLinkClass = (isActive: boolean) =>
    cn(
      'sidebar-nav-link group flex min-h-11 items-center gap-3 rounded-xl px-3 py-2 text-sm font-medium text-base-content/72 transition-all duration-150',
      'hover:text-base-content focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/45',
      collapsed && !isMobile ? 'justify-center px-0' : '',
      isActive && 'sidebar-nav-link-active text-base-content'
    );

  const NavContent = () => (
    <>
      <div className={cn(
        'border-b ghost-divider',
        collapsed && !isMobile ? 'p-3' : 'p-4'
      )}>
        <div className={cn(
          'flex items-center gap-3',
          collapsed && !isMobile ? 'justify-center' : 'justify-between'
        )}>
          {(!collapsed || isMobile) && (
            <div className="flex min-w-0 items-center gap-3">
              <div className="sidebar-brand-mark grid h-11 w-11 shrink-0 place-items-center rounded-xl border text-primary">
                <Blocks className="w-5 h-5" />
              </div>
              <div className="min-w-0">
                <p className="text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-base-content/50">Orchestrator</p>
                <h1 className="truncate text-lg font-bold leading-tight text-base-content">Node-RED</h1>
                <p className="text-xs text-base-content/65">Control Center</p>
              </div>
            </div>
          )}
          {collapsed && !isMobile && (
            <div className="sidebar-brand-mark grid h-10 w-10 place-items-center rounded-xl border text-primary">
              <Box className="w-5 h-5" />
            </div>
          )}
          {!isMobile && (
            <button
              onClick={toggleCollapsed}
              className="sidebar-collapse-button grid h-9 w-9 place-items-center rounded-xl border text-base-content/70 transition-colors hover:text-base-content focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/45"
              aria-label={collapsed ? 'Expandir navegación' : 'Contraer navegación'}
            >
              {collapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
            </button>
          )}
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto px-3 py-4" aria-label="Navegación principal">
        <ul className="space-y-1.5">
          {navItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                className={({ isActive }) => navLinkClass(isActive)}
                title={collapsed && !isMobile ? item.label : undefined}
              >
                <item.icon className="h-5 w-5 flex-shrink-0 stroke-[1.8] text-base-content/55 transition-colors group-hover:text-accent" />
                {(!collapsed || isMobile) && <span className="truncate">{item.label}</span>}
              </NavLink>
            </li>
          ))}

          {isAdmin && (
            <>
              <li className="py-3">
                <div className="h-px bg-[var(--ds-border-default)]" />
              </li>
              {adminItems.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    className={({ isActive }) => navLinkClass(isActive)}
                    title={collapsed && !isMobile ? item.label : undefined}
                  >
                    <item.icon className="h-5 w-5 flex-shrink-0 stroke-[1.8] text-base-content/55 transition-colors group-hover:text-accent" />
                    {(!collapsed || isMobile) && <span className="truncate">{item.label}</span>}
                  </NavLink>
                </li>
              ))}
            </>
          )}
        </ul>
      </nav>

      {user && (
        <div className="border-t ghost-divider p-3">
          <UserMenu user={user} onLogout={handleLogout} />
        </div>
      )}
    </>
  );

  return (
    <aside
      data-testid="app-sidebar"
      className={cn(
        'app-sidebar-shell flex h-full min-h-full flex-col border-r',
        collapsed ? 'w-16' : 'w-64',
        'transition-all duration-200'
      )}
    >
      <NavContent />
    </aside>
  );
}
