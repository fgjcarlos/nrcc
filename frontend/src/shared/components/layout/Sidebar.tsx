import { useState, useEffect } from 'react';
import { NavLink } from 'react-router-dom';
import {
  ActivitySquare,
  Blocks,
  Box,
  ChevronLeft,
  ChevronRight,
  Container,
  DatabaseBackup,
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
  { to: '/runtime', label: 'Runtime', icon: ActivitySquare },
  { to: '/logs', label: 'Logs', icon: FileTerminal },
  { to: '/docker', label: 'Docker', icon: Container },
  { to: '/flows', label: 'Flows', icon: Workflow },
  { to: '/backups', label: 'Backups', icon: DatabaseBackup },
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

  const NavContent = () => (
    <>
      <div className={cn(
        "flex items-center border-b ghost-divider",
        collapsed && !isMobile ? "p-3 justify-center" : "p-5"
      )}>
        {(!collapsed || isMobile) && (
          <div className="flex items-center flex-1 gap-3">
            <div className="grid w-10 h-10 border rounded-lg place-items-center border-primary/25 bg-primary/12 text-primary ">
              <Blocks className="w-5 h-5" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-base-content/50">Orchestrator</p>
              <h1 className="text-lg font-bold leading-tight text-base-content">Node-RED</h1>
              <p className="text-xs text-base-content/65">Control Center</p>
            </div>
          </div>
        )}
        {collapsed && !isMobile && (
          <div className="grid w-10 h-10 border rounded-lg place-items-center border-primary/25 bg-primary/12 text-primary">
            <Box className="w-5 h-5" />
          </div>
        )}
        {!isMobile && (
          <button
            onClick={toggleCollapsed}
            className="rounded-lg btn btn-ghost btn-sm btn-square"
            aria-label={collapsed ? 'Expandir navegación' : 'Contraer navegación'}
          >
            {collapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
          </button>
        )}
      </div>

      <nav className="flex-1 p-3 overflow-y-auto">
        <ul className={cn("menu", collapsed && !isMobile ? "menu-compact" : "menu-md w-full")}>
          {navItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    collapsed && !isMobile ? 'justify-center' : '', 
                    isActive && 'active'
                  )
                }
                title={collapsed && !isMobile ? item.label : undefined}
              >
                <item.icon className="w-5 h-5 flex-shrink-0 stroke-[1.8]" />
                {(!collapsed || isMobile) && <span>{item.label}</span>}
              </NavLink>
            </li>
          ))}

          {isAdmin && (
            <>
              <div className="my-3 divider"></div>
              {adminItems.map((item) => (
                <li key={item.to}>
                  <NavLink
                    to={item.to}
                    className={({ isActive }) =>
                      cn(
                        collapsed && !isMobile ? 'justify-center' : '',
                        isActive && 'active'
                      )
                    }
                    title={collapsed && !isMobile ? item.label : undefined}
                  >
                    <item.icon className="w-5 h-5 flex-shrink-0 stroke-[1.8]" />
                    {(!collapsed || isMobile) && <span>{item.label}</span>}
                  </NavLink>
                </li>
              ))}
            </>
          )}
        </ul>
      </nav>

      {/* User menu at the bottom */}
      {user && (
        <div className="p-4 border-t ghost-divider">
          <UserMenu user={user} onLogout={handleLogout} />
        </div>
      )}
    </>
  );

  return (
    <aside className={cn(
      "glass-panel border-r ghost-divider flex h-full min-h-full flex-col",
      collapsed ? "w-16" : "w-64",
      "transition-all duration-200"
    )}>
      <NavContent />
    </aside>
  );
}
