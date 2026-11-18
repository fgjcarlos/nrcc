import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { ChevronDown, LogOut } from 'lucide-react';
import { User } from '@/features/auth/services/authService';
import { useT } from '@/i18n';
import { cn } from '@/shared/lib/utils';

export interface UserMenuProps {
  user: User;
  onLogout: () => void;
  logoutBusy?: boolean;
  /** Collapsed rail: only the initials show; the name stays in the accessible name. */
  compact?: boolean;
}

// The trigger sits on the ink navigation rail; the open menu is a raised sheet.
export function UserMenu({ user, onLogout, logoutBusy = false, compact = false }: UserMenuProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const { t } = useT();

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

  const handleProfile = () => {
    navigate('/profile');
    setOpen(false);
  };

  return (
    <div ref={containerRef} className="relative">
      {/* Avatar trigger button */}
      {/* The accessible name comes from the visible content (WCAG 2.5.3
          label in name): an author label that omits the rendered initials
          or the role breaks the label-content-name-mismatch rule. The
          action hint is appended as visually hidden text. */}
      <button
        type="button"
        aria-haspopup="true"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        className={cn(
          'rail-control flex min-h-11 w-full items-center gap-3 rounded-field p-2 transition-colors duration-150',
          compact && 'justify-center',
        )}
      >
        <span className="rail-hairline flex h-8 w-8 shrink-0 items-center justify-center rounded-full border font-mono text-xs font-semibold text-[var(--ds-rail-text)]">
          {initials}
        </span>
        {' '}
        <span className={cn('min-w-0 flex-1 text-left', compact && 'sr-only')}>
          <span className="block truncate text-sm font-semibold text-[var(--ds-rail-text)]">{user.username}</span>
          {' '}
          <span className="rail-muted block text-xs capitalize">{user.role}</span>
        </span>
        {' '}
        <span className="sr-only">{t('common:openUserMenu')}</span>
        {!compact && (
          <ChevronDown
            aria-hidden="true"
            className={`h-4 w-4 shrink-0 transition-transform duration-150 ${open ? 'rotate-180' : ''}`}
          />
        )}
      </button>

      {/* Dropdown menu */}
       {open && (
         <div
           role="menu"
           className={cn(
             'absolute bottom-full left-0 mb-2 rounded-box border border-ds-border-default bg-ds-bg-elevated text-base-content shadow-[var(--ds-shadow-raised)]',
             compact ? 'w-56' : 'w-full',
           )}
         >
           <div className="border-b border-ds-border-default px-4 py-3">
             <p className="text-xs text-ds-text-muted">{t('common:signedInAs')}</p>
             <p className="mt-0.5 text-sm font-semibold text-base-content">{user.username}</p>
           </div>
           <div className="p-2 space-y-1">
             <button
               type="button"
               role="menuitem"
               onClick={handleProfile}
               className="flex min-h-11 w-full items-center gap-2 rounded-field px-3 text-sm text-ds-text-secondary transition-colors duration-150 hover:bg-base-200 hover:text-base-content"
             >
               {t('common:profile')}
             </button>
             {/* Logout button */}
             <button
               type="button"
               role="menuitem"
               onClick={handleLogout}
               disabled={logoutBusy}
               className="flex min-h-11 w-full items-center gap-2 rounded-field px-3 text-sm text-ds-text-secondary transition-colors duration-150 hover:bg-base-200 hover:text-base-content disabled:opacity-50 disabled:cursor-not-allowed"
             >
               <LogOut className="w-4 h-4 shrink-0" aria-hidden="true" />
               <span>{logoutBusy ? t('common:signingOut') : t('common:signOut')}</span>
             </button>
           </div>
        </div>
      )}
    </div>
  );
}
