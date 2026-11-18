import { Menu } from 'lucide-react';
import { ThemeToggle } from '@/shared/components/ThemeToggle';
import { NrccMark } from '@/shared/components/NrccMark';
import { UpdateNotificationChip } from '@/features/updates/components/UpdateNotificationChip';
import { CommandPalette } from '@/shared/components/command-palette';
import { useAppChrome } from '@/features/system/hooks/useAppChrome';
import { RuntimeStatusPill } from '@/features/system/components/RuntimeStatusPill';
import { LocaleSwitcher, useT } from '@/i18n';

export interface HeaderProps {
  /** Below `lg`: show the menu button and the brand (the rail is a drawer). */
  compact?: boolean;
  navOpen?: boolean;
  onOpenNav?: () => void;
}

/**
 * Header: the persistent top bar. One runtime status (the same host
 * status Configuration shows) plus the app controls. The brand appears
 * here only when the rail is hidden, so "NRCC" shows once per screen.
 * Below `md` the status wraps onto its own row so it stays visible.
 */
export function Header({ compact = false, navOpen = false, onOpenNav }: HeaderProps) {
  const { t } = useT();
  const chrome = useAppChrome();

  return (
    <header
      role="banner"
      data-testid="app-topbar"
      className="app-topbar-shell sticky top-0 z-40 border-b px-3 py-2 sm:px-5"
    >
      <div className="flex flex-wrap items-center gap-2 md:flex-nowrap md:gap-3">
        {compact && (
          <div className="flex min-w-0 items-center gap-2">
            <button
              type="button"
              onClick={onOpenNav}
              aria-expanded={navOpen}
              aria-haspopup="dialog"
              aria-label={t('common:layout.openNavigation')}
              className="grid h-11 w-11 shrink-0 place-items-center rounded-field border border-ds-border-strong bg-ds-bg-surface text-ds-text-primary hover:bg-base-200"
            >
              <Menu className="h-5 w-5" aria-hidden="true" />
            </button>
            <NrccMark tone="accent" size="sm" />
            <span className="text-sm font-bold tracking-wide text-ds-text-primary">NRCC</span>
          </div>
        )}

        <div className="order-last w-full md:order-none md:w-auto">
          <RuntimeStatusPill {...chrome} />
        </div>

        <div className="ml-auto flex shrink-0 items-center gap-2">
          <CommandPalette />
          <UpdateNotificationChip />
          <LocaleSwitcher />
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
