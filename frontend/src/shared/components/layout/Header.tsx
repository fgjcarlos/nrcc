import { Menu } from 'lucide-react';
import { StatusChip, ThemeToggle, NrccMark } from '@/shared/components';
import { UpdateNotificationChip } from '@/features/updates/components/UpdateNotificationChip';
import { CommandPalette } from '@/shared/components/command-palette';
import { useAppChrome } from '@/features/system/hooks/useAppChrome';
import { NodeRedVersionChip } from '@/features/system/components/NodeRedVersionChip';
import { CompatModeChip } from '@/features/system/components/CompatModeChip';
import { LocaleSwitcher, useT } from '@/i18n';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001';

/**
 * Header — issue #766 slice C
 *
 * The persistent topbar. Slice C turns the right-hand control strip
 * into a coherent "runtime context" panel: detected Node-RED version,
 * configuration mode (editable/read-only), edge deployment flag, and
 * the API base URL. Every chip is built on the StatusChip surface
 * landed in slice B; nothing in this file invents a new visual
 * primitive.
 *
 * The chips are read from useAppChrome(), which refetches
 * /api/system/info and /api/bootstrap/status at the dashboard's own
 * cadence (10s and 30s respectively). Fields that have not yet
 * resolved render their neutral "loading" state.
 */
export function Header() {
  const { t } = useT();
  const chrome = useAppChrome();
  const apiHost = (() => {
    try {
      return new URL(API_URL).host || API_URL;
    } catch {
      return API_URL;
    }
  })();

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
            <NrccMark tone="accent" size="md" />
            <div className="min-w-0 leading-tight">
              <span className="block text-[0.68rem] font-semibold uppercase tracking-[0.24em] text-base-content/55">{t('common:productShortName')}</span>
              <span className="block truncate text-sm font-semibold text-base-content sm:text-base">{t('common:productFullName')}</span>
            </div>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2 sm:gap-3">
          {/* Runtime context chips — slice C */}
          <div
            className="hidden items-center gap-1.5 md:flex"
            data-testid="app-runtime-context"
            aria-label={t('common:appChrome.nodeRedVersionLabel')}
          >
            <NodeRedVersionChip version={chrome.nodeRedVersion} />
            <CompatModeChip
              editable={chrome.configurationEditable}
              mode={chrome.configurationMode}
              runtimeVersion={chrome.configurationRuntimeVersion ?? chrome.nodeRedVersion}
              reason={chrome.configurationReason}
            />
            <StatusChip
              variant={chrome.edgeMode ? 'info' : 'neutral'}
              size="sm"
              ariaLabel={t('common:appChrome.edgeModeAria', {
                state: chrome.edgeMode ? 'on' : 'off',
              })}
              data-testid="edge-mode-chip"
            >
              {chrome.edgeMode
                ? t('common:appChrome.edgeModeEnabled')
                : t('common:appChrome.edgeModeDisabled')}
            </StatusChip>
            <StatusChip
              variant="neutral"
              size="sm"
              ariaLabel={t('common:appChrome.apiBaseAria', { url: apiHost })}
              data-testid="api-base-chip"
              className="hidden xl:inline-flex"
            >
              {t('common:appChrome.apiBaseLabel')}: {apiHost}
            </StatusChip>
          </div>

          {/* Application chrome — existing controls */}
          <CommandPalette />
          <UpdateNotificationChip />
          <LocaleSwitcher />
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
