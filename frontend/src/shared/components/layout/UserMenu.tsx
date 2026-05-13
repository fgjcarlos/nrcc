import { useState, useRef, useEffect } from 'react';
import { ChevronDown, LogOut } from 'lucide-react';
import { User } from '@/features/auth/services/authService';

export interface UserMenuProps {
  user: User;
  onLogout: () => void;
  logoutBusy?: boolean;
}

const AVATAR_COLORS = [
  { bg: 'bg-red-500/20', text: 'text-red-400' },
  { bg: 'bg-violet-500/20', text: 'text-violet-400' },
  { bg: 'bg-sky-500/20', text: 'text-sky-400' },
  { bg: 'bg-emerald-500/20', text: 'text-emerald-400' },
  { bg: 'bg-amber-500/20', text: 'text-amber-400' },
  { bg: 'bg-rose-500/20', text: 'text-rose-400' },
];

export function UserMenu({ user, onLogout, logoutBusy = false }: UserMenuProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Get deterministic avatar color from username
  const avatarColorIndex = user.username.charCodeAt(0) % AVATAR_COLORS.length;
  const avatarColor = AVATAR_COLORS[avatarColorIndex];
  const initials = user.username.slice(0, 2).toUpperCase();

  // Handle Escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && open) {
        setOpen(false);
      }
    };

    if (open) {
      document.addEventListener('keydown', handleKeyDown);
      return () => document.removeEventListener('keydown', handleKeyDown);
    }
  }, [open]);

  // Handle outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };

    if (open) {
      document.addEventListener('mousedown', handleClickOutside);
      return () => document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [open]);

  const handleLogout = () => {
    onLogout();
    setOpen(false);
  };

  return (
    <div ref={containerRef} className="relative">
      {/* Avatar trigger button */}
      <button
        type="button"
        aria-haspopup="true"
        aria-expanded={open}
        aria-label={`${user.username} — open user menu`}
        onClick={() => setOpen(!open)}
        className="flex w-full items-center gap-3 rounded-xl p-3 hover:bg-base-300/50 transition-colors duration-150"
      >
        <span className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold ${avatarColor.bg} ${avatarColor.text}`}>
          {initials}
        </span>
        <span className="min-w-0 flex-1 text-left">
          <span className="block truncate text-sm font-semibold text-base-content">{user.username}</span>
          <span className="block text-xs text-base-content/50 capitalize">{user.role}</span>
        </span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-base-content/40 transition-transform duration-150 ${open ? 'rotate-180' : ''}`}
        />
      </button>

      {/* Dropdown menu */}
      {open && (
        <div
          role="menu"
          className="absolute bottom-full left-0 mb-2 w-full rounded-xl border border-base-300/60 bg-base-200 shadow-lg transition-all duration-150 origin-bottom animate-slide-up"
        >
          <div className="border-b border-base-300/40 px-4 py-3">
            <p className="text-xs text-base-content/50 uppercase tracking-widest">Signed in as</p>
            <p className="mt-0.5 font-semibold text-base-content">{user.username}</p>
          </div>
           <div className="p-2 space-y-1">
             {/* Profile item (disabled for now) */}
             <button
               type="button"
               role="menuitem"
               disabled
               className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-base-content/50 cursor-not-allowed transition-colors duration-150"
               title="Profile page coming soon"
             >
               Perfil
             </button>
             {/* Logout button */}
             <button
               type="button"
               role="menuitem"
               onClick={handleLogout}
               disabled={logoutBusy}
               className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-base-content/70 transition-colors duration-150 hover:bg-error/15 hover:text-error/90 disabled:opacity-50 disabled:cursor-not-allowed"
             >
               <LogOut className="w-4 h-4 shrink-0" />
               <span>{logoutBusy ? 'Signing out…' : 'Cerrar sesión'}</span>
             </button>
           </div>
        </div>
      )}
    </div>
  );
}
