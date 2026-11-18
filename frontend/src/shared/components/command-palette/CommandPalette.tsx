import { FormEvent, KeyboardEvent, useEffect, useMemo, useRef, useState } from 'react';
import { Search } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { backupService } from '@/features/backups/services';
import { dashboardService } from '@/features/dashboard/services';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { queryKeys } from '@/shared/lib/queryKeys';
import { cn } from '@/shared/lib/utils';
import { useT } from '@/i18n';
import { useModalKeyboard } from '@/shared/hooks/useModalKeyboard';

type CommandKind = 'navigation' | 'service' | 'external';

type Command = {
  id: string;
  title: string;
  description: string;
  keywords: string[];
  kind: CommandKind;
  adminOnly?: boolean;
  confirmMessage?: string;
  run: () => Promise<void> | void;
};

const nodeRedEditorPort = import.meta.env.VITE_NODE_RED_PORT || '1880';

function commandMatches(command: Command, query: string) {
  const haystack = [command.title, command.description, command.kind, ...command.keywords].join(' ').toLowerCase();
  return haystack.includes(query.trim().toLowerCase());
}

export function CommandPalette() {
  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);
  const [isExecuting, setIsExecuting] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const { t } = useT();
  const queryClient = useQueryClient();
  const { data: configResponse } = useQuery({
    queryKey: queryKeys.config.root,
    queryFn: () => dashboardService.getConfig(),
    enabled: isOpen,
    staleTime: 60_000,
  });
  const { user } = useAuth();
  const isAdmin = user?.role === 'admin';
  const uiPort = configResponse?.data?.data?.uiPort ?? nodeRedEditorPort;

  const closePalette = () => {
    setIsOpen(false);
    setQuery('');
    setActiveIndex(0);
  };

  const commands = useMemo<Command[]>(() => {
    // Navigation titles reuse each page's h1 key, like the sidebar.
    const go = (id: string, page: string, path: string, keywords: string[], adminOnly?: boolean): Command => ({
      id: `nav-${id}`,
      title: t('common:palette.goTo', { page: t(page) }),
      description: t(`common:palette.describe.${id}`),
      keywords,
      kind: 'navigation',
      adminOnly,
      run: () => navigate(path),
    });
    const service = (id: string, keywords: string[], run: () => Promise<void>): Command => ({
      id,
      title: t(`common:palette.${id}.title`),
      description: t(`common:palette.${id}.description`),
      keywords,
      kind: 'service',
      adminOnly: true,
      confirmMessage: t(`common:palette.${id}.confirm`),
      run,
    });
    const refreshRuntime = () => queryClient.invalidateQueries({ queryKey: queryKeys.runtime.status });

    return [
      go('overview', 'dashboard:overview', '/overview', ['home', 'status', 'runtime', 'overview']),
      go('configuration', 'configuration:pageTitle', '/configuration', ['settings', 'config', 'node-red']),
      go('updates', 'updates:title', '/maintenance/updates', ['updates', 'upgrade', 'release', 'version', 'maintenance'], true),
      go('libraries', 'libraries:title', '/maintenance/libraries', ['libraries', 'nodes', 'packages', 'palette', 'maintenance'], true),
      go('bootstrap', 'bootstrap:bootstrapTitle', '/bootstrap', ['setup', 'host', 'prerequisites', 'bootstrap']),
      go('environment', 'env-vars:pageTitle', '/environment', ['env', 'variables', 'secrets']),
      go('backups', 'backups:backupsTitle', '/backups', ['archive', 'restore', 'snapshot', 'recovery']),
      go('profile', 'common:profile', '/profile', ['account', 'password', 'user']),
      go('users', 'auth:userManagement', '/settings/users', ['admin', 'permissions', 'roles'], true),
      service('restart', ['service', 'runtime', 'reload'], async () => {
        await dashboardService.restartNodeRed();
        void refreshRuntime();
      }),
      service('start', ['service', 'runtime', 'up'], async () => {
        await dashboardService.startNodeRed();
        void refreshRuntime();
      }),
      service('stop', ['service', 'runtime', 'down'], async () => {
        await dashboardService.stopNodeRed();
        void refreshRuntime();
      }),
      service('backupNow', ['snapshot', 'archive', 'manual'], async () => {
        await backupService.create('manual');
        queryClient.invalidateQueries({ queryKey: queryKeys.backups.listRoot });
        queryClient.invalidateQueries({ queryKey: queryKeys.backups.status });
        queryClient.invalidateQueries({ queryKey: queryKeys.backups.observability });
      }),
      {
        id: 'open-node-red-editor',
        title: t('common:palette.openEditor.title'),
        description: t('common:palette.openEditor.description'),
        keywords: ['editor', 'external', 'localhost'],
        kind: 'external',
        run: () => {
          // Validate uiPort is an integer in the valid TCP port range
          // before constructing the loopback URL. Defends against
          // misconfigured backend responses or env injection.
          const parsed = Number.parseInt(String(uiPort), 10);
          const safePort = Number.isInteger(parsed) && parsed >= 1 && parsed <= 65535 ? parsed : 1880;
          window.open(`http://localhost:${safePort}`, '_blank', 'noopener,noreferrer');
        },
      },
    ];
  }, [navigate, queryClient, uiPort, t]);

  const availableCommands = useMemo(
    () => commands.filter((command) => !command.adminOnly || isAdmin),
    [commands, isAdmin],
  );

  const filteredCommands = useMemo(() => {
    const normalizedQuery = query.trim();
    if (!normalizedQuery) {
      return availableCommands;
    }
    return availableCommands.filter((command) => commandMatches(command, normalizedQuery));
  }, [availableCommands, query]);

  useEffect(() => {
    const onKeyDown = (event: globalThis.KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setIsOpen((current) => !current);
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  // Focus the search on open, trap Tab, close on Escape, return focus.
  useModalKeyboard({ isOpen, containerRef: dialogRef, onEscape: closePalette, initialFocusRef: inputRef });

  useEffect(() => {
    setActiveIndex(0);
  }, [query]);

  useEffect(() => {
    if (activeIndex >= filteredCommands.length) {
      setActiveIndex(Math.max(filteredCommands.length - 1, 0));
    }
  }, [activeIndex, filteredCommands.length]);

  const executeCommand = async (command: Command) => {
    if (command.confirmMessage && !window.confirm(command.confirmMessage)) {
      return;
    }

    setIsExecuting(true);
    try {
      await command.run();
      toast.success(t('common:palette.executed'), { description: command.title });
      closePalette();
    } catch (error) {
      toast.error(t('common:palette.failed'), {
        description:
          error instanceof Error ? error.message : t('common:palette.failedFallback'),
        duration: 8000,
      });
    } finally {
      setIsExecuting(false);
    }
  };

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    const command = filteredCommands[activeIndex];
    if (command && !isExecuting) {
      void executeCommand(command);
    }
  };

  const onInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      setActiveIndex((current) => (filteredCommands.length ? (current + 1) % filteredCommands.length : 0));
      return;
    }

    if (event.key === 'ArrowUp') {
      event.preventDefault();
      setActiveIndex((current) => (filteredCommands.length ? (current - 1 + filteredCommands.length) % filteredCommands.length : 0));
    }
  };

  return (
    <>
      <button
        type="button"
        className="inline-flex h-11 min-w-11 items-center justify-center gap-2 rounded-field border border-ds-border-strong bg-ds-bg-surface px-3 text-sm font-medium text-ds-text-primary transition-colors hover:bg-base-200"
        onClick={() => setIsOpen(true)}
        aria-haspopup="dialog"
        aria-expanded={isOpen}
      >
        <Search className="h-5 w-5 text-ds-text-secondary lg:hidden" aria-hidden="true" />
        <span className="sr-only lg:not-sr-only">{t('common:commandPalette')}</span>
        <kbd className="hidden rounded-selector border border-ds-border-default bg-ds-bg-base px-1.5 py-0.5 font-mono text-xs text-ds-text-muted lg:inline">{t('common:commandPaletteShortcut')}</kbd>
      </button>

      {isOpen && (
        <div className="motion-scrim fixed inset-0 z-[70] bg-[color-mix(in_srgb,var(--ds-rail-bg)_55%,transparent)] p-4" role="presentation" onMouseDown={closePalette}>
          <div
            ref={dialogRef}
            role="dialog"
            aria-modal="true"
            aria-label={t('common:commandPalette')}
            className="motion-overlay mx-auto mt-16 w-full max-w-2xl overflow-hidden rounded-box border border-ds-border-default bg-ds-bg-elevated shadow-[var(--ds-shadow-raised)]"
            onMouseDown={(event) => event.stopPropagation()}
          >
            <form onSubmit={onSubmit}>
              <label className="sr-only" htmlFor="command-palette-search">{t('common:searchCommands')}</label>
              <input
                ref={inputRef}
                id="command-palette-search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                onKeyDown={onInputKeyDown}
                placeholder={t('common:palette.searchPlaceholder')}
                // input-bare: drawn into the palette, so it keeps its rule line instead of the field box.
                className="input-bare w-full border-b border-ds-border-default bg-transparent px-5 py-4 text-sm text-base-content outline-none placeholder:text-ds-text-muted focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
                role="combobox"
                aria-expanded="true"
                aria-controls="command-palette-results"
                aria-activedescendant={filteredCommands[activeIndex]?.id}
              />

              <div id="command-palette-results" role="listbox" className="max-h-[28rem] overflow-y-auto p-2">
                {filteredCommands.length === 0 ? (
                  <div className="px-4 py-8 text-center text-sm text-ds-text-muted">{t('common:noMatchingCommands')}</div>
                ) : (
                  filteredCommands.map((command, index) => (
                    <button
                      id={command.id}
                      key={command.id}
                      type="button"
                      role="option"
                      aria-selected={index === activeIndex}
                      className={cn(
                        'flex w-full items-start justify-between gap-4 rounded-field px-4 py-3 text-left transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
                        index === activeIndex ? 'bg-ds-accent-dim text-ds-text-primary' : 'text-ds-text-primary hover:bg-base-200',
                      )}
                      onMouseEnter={() => setActiveIndex(index)}
                      onClick={() => void executeCommand(command)}
                      disabled={isExecuting}
                    >
                      <span>
                        <span className="block text-sm font-semibold">{command.title}</span>
                        <span className="mt-1 block text-xs text-ds-text-secondary">{command.description}</span>
                      </span>
                      <span className="shrink-0 rounded-selector border border-ds-border-default px-2 py-0.5 text-xs text-ds-text-secondary">
                        {t(`common:palette.kind.${command.kind}`)}
                      </span>
                    </button>
                  ))
                )}
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
