import { useEffect, useState, type ComponentType } from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import {
  ChevronLeft,
  ChevronRight,
  DatabaseBackup,
  Gauge,
  HardDrive,
  Library,
  RotateCw,
  Settings,
  Shield,
  SlidersHorizontal,
  UsersRound,
  X,
} from 'lucide-react';
import { cn } from '@/shared/lib/utils';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { useT } from '@/i18n';
import { NrccMark } from '@/shared/components/NrccMark';
import { UserMenu } from './UserMenu';

type NavItemDef = { to: string; labelKey: string; icon: ComponentType<{ className?: string }> };

// Each label is its page's h1 key, so nav and heading cannot drift apart.
const navItems: NavItemDef[] = [
  { to: '/overview', labelKey: 'dashboard:overview', icon: Gauge },
  { to: '/configuration', labelKey: 'configuration:pageTitle', icon: Settings },
  { to: '/security', labelKey: 'security:title', icon: Shield },
  { to: '/environment', labelKey: 'env-vars:pageTitle', icon: SlidersHorizontal },
  { to: '/backups', labelKey: 'backups:backupsTitle', icon: DatabaseBackup },
  { to: '/bootstrap', labelKey: 'bootstrap:bootstrapTitle', icon: HardDrive },
];

const adminItems: NavItemDef[] = [
  { to: '/settings/users', labelKey: 'auth:userManagement', icon: UsersRound },
  { to: '/maintenance/updates', labelKey: 'updates:title', icon: RotateCw },
  { to: '/maintenance/libraries', labelKey: 'libraries:title', icon: Library },
];

const STORAGE_KEY = 'sidebar-collapsed';
const ADMIN_GROUP_ID = 'nav-admin-group';

function NavItem({ item, iconOnly, onNavigate }: { item: NavItemDef; iconOnly: boolean; onNavigate?: () => void }) {
  const { t } = useT();
  const label = t(item.labelKey);
  return (
    <li>
      <NavLink
        to={item.to}
        onClick={onNavigate}
        title={iconOnly ? label : undefined}
        aria-label={iconOnly ? label : undefined}
        className={({ isActive }) =>
          cn(
            'rail-control flex min-h-11 items-center gap-3 rounded-field px-3 text-sm font-medium transition-colors duration-150',
            iconOnly && 'justify-center px-0',
            isActive && 'sidebar-nav-link-active font-semibold',
          )
        }
      >
        <item.icon className="h-5 w-5 shrink-0 stroke-[1.8]" aria-hidden="true" />
        {!iconOnly && <span className="truncate">{label}</span>}
      </NavLink>
    </li>
  );
}

export interface SidebarProps {
  /** `rail`: persistent, collapsible (lg and up). `drawer`: inside the mobile drawer. */
  mode?: 'rail' | 'drawer';
  /** Drawer mode: closes the drawer (close button, followed link). */
  onClose?: () => void;
}

/**
 * Sidebar: the ink navigation rail (DESIGN.md). Labelled links (icon
 * plus text) that collapse to icons on desktop; admin links sit in one
 * labelled "Administration" group shown only to admins.
 */
export function Sidebar({ mode = 'rail', onClose }: SidebarProps) {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const { t } = useT();
  const isAdmin = user?.role === 'admin';
  const isDrawer = mode === 'drawer';

  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(STORAGE_KEY) === 'true');
  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, String(collapsed));
  }, [collapsed]);

  const iconOnly = collapsed && !isDrawer;

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <aside
      data-testid="app-sidebar"
      className={cn(
        'app-sidebar-shell flex h-full min-h-full flex-col border-r transition-[width] duration-200',
        iconOnly ? 'w-16' : 'w-64',
      )}
    >
      <div className={cn('rail-hairline flex min-h-16 items-center gap-2 border-b', iconOnly ? 'flex-col justify-center p-2' : 'px-3')}>
        <NrccMark tone="rail" size="sm" />
        {!iconOnly && <span className="flex-1 text-sm font-bold tracking-wide">NRCC</span>}
        {isDrawer ? (
          <button
            type="button"
            onClick={onClose}
            className="rail-control grid h-11 w-11 place-items-center rounded-field"
            aria-label={t('common:layout.closeNavigation')}
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        ) : (
          <button
            type="button"
            onClick={() => setCollapsed((current) => !current)}
            className="rail-control grid h-9 w-9 place-items-center rounded-field"
            aria-label={collapsed ? t('common:layout.expandNavigation') : t('common:layout.collapseNavigation')}
            aria-expanded={!collapsed}
          >
            {collapsed ? <ChevronRight className="h-4 w-4" aria-hidden="true" /> : <ChevronLeft className="h-4 w-4" aria-hidden="true" />}
          </button>
        )}
      </div>

      <nav className="flex-1 overflow-y-auto px-2 py-3" aria-label={t('common:layout.mainNavigation')}>
        <ul className="space-y-1">
          {navItems.map((item) => (
            <NavItem key={item.to} item={item} iconOnly={iconOnly} onNavigate={onClose} />
          ))}
        </ul>

        {isAdmin && (
          <div role="group" aria-labelledby={ADMIN_GROUP_ID} className="mt-4">
            <p
              id={ADMIN_GROUP_ID}
              className={cn(
                'rail-muted px-3 pb-1 text-xs font-semibold uppercase tracking-wider',
                iconOnly && 'sr-only',
              )}
            >
              {t('common:layout.administration')}
            </p>
            {iconOnly && <div className="rail-hairline mx-2 mb-2 border-t" aria-hidden="true" />}
            <ul className="space-y-1">
              {adminItems.map((item) => (
                <NavItem key={item.to} item={item} iconOnly={iconOnly} onNavigate={onClose} />
              ))}
            </ul>
          </div>
        )}
      </nav>

      {user && (
        <div className="rail-hairline border-t p-2">
          <UserMenu user={user} onLogout={handleLogout} compact={iconOnly} />
        </div>
      )}
    </aside>
  );
}
