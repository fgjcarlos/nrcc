import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CommandPalette } from './CommandPalette';
import { dashboardService } from '@/features/dashboard/services';
import { backupService } from '@/features/backups/services';

vi.mock('@/features/dashboard/services', () => ({
  dashboardService: {
    restartNodeRed: vi.fn(),
    startNodeRed: vi.fn(),
    stopNodeRed: vi.fn(),
    getConfig: vi.fn(),
  },
}));

vi.mock('@/features/backups/services', () => ({
  backupService: {
    create: vi.fn(),
  },
}));

const authState = vi.hoisted(() => ({
  user: { id: 'u1', username: 'admin', role: 'admin', createdAt: '2024-01-01T00:00:00Z' },
}));

vi.mock('@/features/auth/hooks/useAuth', () => ({
  useAuth: () => ({
    user: authState.user,
    isAuthenticated: true,
    isInitialized: true,
    isLoading: false,
  }),
}));

const toastSpy = vi.hoisted(() => vi.fn());
vi.mock('sonner', () => ({
  toast: Object.assign(toastSpy, {
    success: toastSpy,
    error: toastSpy,
  }),
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}</div>;
}

function renderPalette(initialPath = '/dashboard') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <QueryClientProvider client={queryClient}>
        <CommandPalette />
        <Routes>
          <Route path="*" element={<LocationProbe />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe('CommandPalette', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    authState.user = { id: 'u1', username: 'admin', role: 'admin', createdAt: '2024-01-01T00:00:00Z' };
    vi.mocked(dashboardService.restartNodeRed).mockResolvedValue({} as Awaited<ReturnType<typeof dashboardService.restartNodeRed>>);
    vi.mocked(dashboardService.startNodeRed).mockResolvedValue({} as Awaited<ReturnType<typeof dashboardService.startNodeRed>>);
    vi.mocked(dashboardService.stopNodeRed).mockResolvedValue({} as Awaited<ReturnType<typeof dashboardService.stopNodeRed>>);
    vi.mocked(dashboardService.getConfig).mockResolvedValue({ data: { data: { uiPort: 1880 } } } as Awaited<ReturnType<typeof dashboardService.getConfig>>);
    vi.mocked(backupService.create).mockResolvedValue({
      id: 'backup-1',
      name: 'manual',
      type: 'manual',
      createdAt: '2024-01-01T00:00:00Z',
      triggeredBy: 'test',
      fileCount: 1,
      totalSize: 100,
    });
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    vi.spyOn(window, 'open').mockImplementation(() => null);
  });

  it('opens with Ctrl+K and filters searchable commands', async () => {
    renderPalette();
    const user = userEvent.setup();

    await user.keyboard('{Control>}k{/Control}');

    expect(screen.getByRole('dialog', { name: /command palette/i })).toBeInTheDocument();
    const search = screen.getByRole('combobox', { name: /search commands/i });
    await waitFor(() => expect(search).toHaveFocus());

    await user.type(search, 'configuration');

    expect(screen.getByRole('option', { name: /go to configuration/i })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: /go to dashboard/i })).not.toBeInTheDocument();
  });

  it('does not offer navigation to removed Logs or Docker pages', async () => {
    renderPalette();
    const user = userEvent.setup();

    await user.keyboard('{Control>}k{/Control}');

    expect(screen.queryByRole('option', { name: 'Open Logs' })).not.toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Go to Docker' })).not.toBeInTheDocument();
  });

  it('executes route navigation from the keyboard', async () => {
    renderPalette('/dashboard');
    const user = userEvent.setup();

    await user.keyboard('{Control>}k{/Control}');
    await user.type(screen.getByRole('combobox', { name: /search commands/i }), 'backups');
    await user.keyboard('{Enter}');

    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent('/backups'));
    expect(screen.queryByRole('dialog', { name: /command palette/i })).not.toBeInTheDocument();
  });

  it('confirms and runs protected service commands for admins', async () => {
    renderPalette();
    const user = userEvent.setup();

    await user.keyboard('{Control>}k{/Control}');
    await user.type(screen.getByRole('combobox', { name: /search commands/i }), 'backup now');
    await user.keyboard('{Enter}');

    await waitFor(() => expect(window.confirm).toHaveBeenCalledWith('Create a manual backup now?'));
    await waitFor(() => expect(backupService.create).toHaveBeenCalledWith('manual'));
    expect(toastSpy).toHaveBeenCalledWith('Command executed', expect.objectContaining({ description: 'Backup Now' }));
  });

  it('hides admin-only service commands from viewers', async () => {
    authState.user = { id: 'u2', username: 'viewer', role: 'viewer', createdAt: '2024-01-01T00:00:00Z' };
    renderPalette();
    const user = userEvent.setup();

    await user.keyboard('{Control>}k{/Control}');
    await user.type(screen.getByRole('combobox', { name: /search commands/i }), 'restart');

    expect(screen.queryByRole('option', { name: /restart node-red/i })).not.toBeInTheDocument();
    expect(screen.getByText(/no matching commands/i)).toBeInTheDocument();
  });
});
