import { FormEvent, KeyboardEvent, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { backupService } from '@/features/backups/services';
import { dashboardService } from '@/features/dashboard/services';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { queryKeys } from '@/shared/lib/queryKeys';
import { cn } from '@/shared/lib/utils';

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
  const navigate = useNavigate();
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

  const commands = useMemo<Command[]>(
    () => [
      {
        id: 'nav-dashboard',
        title: 'Go to Dashboard',
        description: 'Open runtime health, host metrics, and quick actions.',
        keywords: ['home', 'status', 'runtime', 'overview'],
        kind: 'navigation',
        run: () => navigate('/dashboard'),
      },
      {
        id: 'nav-configuration',
        title: 'Go to Configuration',
        description: 'Open Node-RED configuration settings.',
        keywords: ['settings', 'config', 'node-red'],
        kind: 'navigation',
        run: () => navigate('/configuration'),
      },
      {
        id: 'nav-logs',
        title: 'Open Logs',
        description: 'Review Node-RED runtime logs.',
        keywords: ['runtime', 'events', 'log'],
        kind: 'navigation',
        run: () => navigate('/logs'),
      },
      {
        id: 'nav-docker',
        title: 'Go to Docker',
        description: 'Open container status and Docker controls.',
        keywords: ['container', 'image', 'compose'],
        kind: 'navigation',
        run: () => navigate('/docker'),
      },
      {
        id: 'nav-updates',
        title: 'Go to Updates',
        description: 'Check and apply NRCC updates.',
        keywords: ['upgrade', 'release', 'version'],
        kind: 'navigation',
        run: () => navigate('/updates'),
      },
      {
        id: 'nav-libraries',
        title: 'Go to Libraries',
        description: 'Manage installed Node-RED libraries.',
        keywords: ['nodes', 'packages', 'palette'],
        kind: 'navigation',
        run: () => navigate('/libraries'),
      },
      {
        id: 'nav-flows',
        title: 'Go to Flows',
        description: 'Open flow inventory and analysis.',
        keywords: ['flows', 'nodes', 'analysis'],
        kind: 'navigation',
        run: () => navigate('/flows'),
      },
      {
        id: 'nav-flow-versions',
        title: 'Go to Flow Versions',
        description: 'Review saved flow version history.',
        keywords: ['history', 'versions', 'backup'],
        kind: 'navigation',
        run: () => navigate('/flows/versions'),
      },
      {
        id: 'nav-bootstrap',
        title: 'Go to Bootstrap',
        description: 'Open host prerequisites and setup status.',
        keywords: ['setup', 'host', 'prerequisites'],
        kind: 'navigation',
        run: () => navigate('/bootstrap'),
      },
      {
        id: 'nav-environment',
        title: 'Go to Environment Variables',
        description: 'Manage runtime environment values.',
        keywords: ['env', 'variables', 'secrets'],
        kind: 'navigation',
        run: () => navigate('/environment'),
      },
      {
        id: 'nav-backups',
        title: 'Go to Backups',
        description: 'Open backup history, restore options, and schedule settings.',
        keywords: ['archive', 'restore', 'snapshot'],
        kind: 'navigation',
        run: () => navigate('/backups'),
      },
      {
        id: 'nav-profile',
        title: 'Go to Profile',
        description: 'Open your account profile.',
        keywords: ['account', 'password', 'user'],
        kind: 'navigation',
        run: () => navigate('/profile'),
      },
      {
        id: 'nav-users',
        title: 'Go to User Management',
        description: 'Manage NRCC users and roles.',
        keywords: ['admin', 'permissions', 'roles'],
        kind: 'navigation',
        adminOnly: true,
        run: () => navigate('/settings/users'),
      },
      {
        id: 'runtime-restart',
        title: 'Restart Node-RED',
        description: 'Restart the Node-RED runtime service.',
        keywords: ['service', 'runtime', 'reload'],
        kind: 'service',
        adminOnly: true,
        confirmMessage: 'Restart Node-RED now? Active flows may be briefly interrupted.',
        run: async () => {
          await dashboardService.restartNodeRed();
          queryClient.invalidateQueries({ queryKey: queryKeys.runtime.status });
        },
      },
      {
        id: 'runtime-start',
        title: 'Start Node-RED',
        description: 'Start the Node-RED runtime service.',
        keywords: ['service', 'runtime', 'up'],
        kind: 'service',
        adminOnly: true,
        confirmMessage: 'Start Node-RED now?',
        run: async () => {
          await dashboardService.startNodeRed();
          queryClient.invalidateQueries({ queryKey: queryKeys.runtime.status });
        },
      },
      {
        id: 'runtime-stop',
        title: 'Stop Node-RED',
        description: 'Stop the Node-RED runtime service.',
        keywords: ['service', 'runtime', 'down'],
        kind: 'service',
        adminOnly: true,
        confirmMessage: 'Stop Node-RED now? Automation flows will stop running.',
        run: async () => {
          await dashboardService.stopNodeRed();
          queryClient.invalidateQueries({ queryKey: queryKeys.runtime.status });
        },
      },
      {
        id: 'backup-now',
        title: 'Backup Now',
        description: 'Create a manual backup immediately.',
        keywords: ['snapshot', 'archive', 'manual'],
        kind: 'service',
        adminOnly: true,
        confirmMessage: 'Create a manual backup now?',
        run: async () => {
          await backupService.create('manual');
          queryClient.invalidateQueries({ queryKey: queryKeys.backups.listRoot });
          queryClient.invalidateQueries({ queryKey: queryKeys.backups.status });
          queryClient.invalidateQueries({ queryKey: queryKeys.backups.observability });
        },
      },
      {
        id: 'open-node-red-editor',
        title: 'Open Node-RED Editor',
        description: 'Open the local Node-RED editor in a new tab.',
        keywords: ['editor', 'external', 'localhost'],
        kind: 'external',
        run: () => {
          window.open(`http://localhost:${uiPort}`, '_blank', 'noopener,noreferrer');
        },
      },
    ],
    [navigate, queryClient, uiPort],
  );

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

  useEffect(() => {
    if (isOpen) {
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [isOpen]);

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
      toast.success('Command executed', { description: command.title });
      closePalette();
    } catch (error) {
      toast.error('Command failed', {
        description:
          error instanceof Error ? error.message : 'Unable to complete the command.',
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
    if (event.key === 'Escape') {
      event.preventDefault();
      closePalette();
      return;
    }

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
        className="hidden rounded-xl border border-border/70 bg-base-300/45 px-3 py-2 text-sm font-medium text-base-content/75 transition hover:border-primary/60 hover:text-base-content focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary sm:inline-flex sm:items-center sm:gap-2"
        onClick={() => setIsOpen(true)}
        aria-haspopup="dialog"
        aria-expanded={isOpen}
      >
        <span>Command palette</span>
        <kbd className="rounded-md border border-border/70 bg-base-100 px-1.5 py-0.5 text-[0.65rem] text-base-content/60">⌘/Ctrl K</kbd>
      </button>

      {isOpen && (
        <div className="fixed inset-0 z-[70] bg-neutral/60 p-4 backdrop-blur-sm" role="presentation" onMouseDown={closePalette}>
          <div
            role="dialog"
            aria-modal="true"
            aria-label="Command palette"
            className="mx-auto mt-16 w-full max-w-2xl overflow-hidden rounded-2xl border border-border bg-base-100 shadow-2xl"
            onMouseDown={(event) => event.stopPropagation()}
          >
            <form onSubmit={onSubmit}>
              <label className="sr-only" htmlFor="command-palette-search">Search commands</label>
              <input
                ref={inputRef}
                id="command-palette-search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                onKeyDown={onInputKeyDown}
                placeholder="Search routes and actions…"
                className="w-full border-b border-border bg-transparent px-5 py-4 text-base text-base-content outline-none placeholder:text-base-content/45 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary"
                role="combobox"
                aria-expanded="true"
                aria-controls="command-palette-results"
                aria-activedescendant={filteredCommands[activeIndex]?.id}
              />

              <div id="command-palette-results" role="listbox" className="max-h-[28rem] overflow-y-auto p-2">
                {filteredCommands.length === 0 ? (
                  <div className="px-4 py-8 text-center text-sm text-base-content/60">No matching commands</div>
                ) : (
                  filteredCommands.map((command, index) => (
                    <button
                      id={command.id}
                      key={command.id}
                      type="button"
                      role="option"
                      aria-selected={index === activeIndex}
                      className={cn(
                        'flex w-full items-start justify-between gap-4 rounded-xl px-4 py-3 text-left transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary',
                        index === activeIndex ? 'bg-primary/12 text-base-content' : 'hover:bg-base-200 text-base-content/85',
                      )}
                      onMouseEnter={() => setActiveIndex(index)}
                      onClick={() => void executeCommand(command)}
                      disabled={isExecuting}
                    >
                      <span>
                        <span className="block font-semibold">{command.title}</span>
                        <span className="mt-1 block text-sm text-base-content/60">{command.description}</span>
                      </span>
                      <span className="shrink-0 rounded-full border border-border px-2 py-1 text-[0.65rem] uppercase tracking-wide text-base-content/55">
                        {command.kind}
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
