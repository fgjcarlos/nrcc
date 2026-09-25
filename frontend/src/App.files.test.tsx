import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import App from './App';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { buildAuthMock } from '@/features/auth/__test-utils__/authMock';
import { authService } from '@/features/auth/services/authService';

vi.mock('@/features/auth/hooks/useAuth', () => ({ useAuth: vi.fn() }));
vi.mock('@/features/auth/services/authService', () => ({ authService: { getStatus: vi.fn() } }));
vi.mock('@/features/auth/components/LoginView', () => ({ LoginView: () => <div>Login page</div> }));
vi.mock('@/features/auth/components/SetupView', () => ({ SetupView: () => <div>Setup page</div> }));
vi.mock('@/features/dashboard/components/DashboardView', () => ({ DashboardView: () => <div>Overview page</div> }));
vi.mock('@/features/backups/components/BackupsView', () => ({ BackupsView: () => <div>Recovery page</div> }));
vi.mock('@/features/updates/components/UpdatesView', () => ({ UpdatesView: () => <div>Updates page</div> }));
vi.mock('@/features/libraries/components/LibrariesView', () => ({ LibrariesView: () => <div>Libraries page</div> }));
vi.mock('@/shared/components', async () => ({
  ...(await vi.importActual<typeof import('@/shared/components')>('@/shared/components')),
  ThemeToggle: () => null,
}));
vi.mock('@/features/updates/components/UpdateNotificationChip', () => ({ UpdateNotificationChip: () => null }));
vi.mock('@/shared/components/command-palette', () => ({ CommandPalette: () => null }));

function mockAuth(overrides: Parameters<typeof buildAuthMock>[0]) {
  vi.mocked(useAuth).mockReturnValue(buildAuthMock(overrides));
}

describe('navigation redirects', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // jsdom does not implement window.matchMedia. The ThemeToggle
    // component (partially mocked above to return null) still runs a
    // useEffect that queries window.matchMedia when @/shared/components
    // is loaded for the partial mock — the real module is loaded
    // transiently through vi.importActual and ThemeToggle's body executes
    // once before the mock applies. Stub matchMedia so the effect
    // completes without throwing. Mirrors the setup in
    // App.sidebar.test.tsx, LayoutShell.test.tsx, and LandingView.test.tsx.
    window.matchMedia = vi.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));
  });

  it.each([
    ['fresh server', false, false, 'Setup page'],
    ['initialized unauthenticated server', true, false, 'Login page'],
    ['initialized authenticated server', true, true, 'Overview page'],
  ])('routes / using the server initialization signal for %s', async (_state, initialized, isAuthenticated, expectedPage) => {
    window.history.pushState({}, '', '/');
    vi.mocked(authService.getStatus).mockResolvedValue({ initialized });
    mockAuth({ isInitialized: true, isAuthenticated, isLoading: false, user: isAuthenticated ? undefined : null });

    render(<App />);

    expect(await screen.findByText(expectedPage)).toBeInTheDocument();
    expect(authService.getStatus).toHaveBeenCalledTimes(1);
  });

  it('routes to login when the server initialization status cannot be read', async () => {
    window.history.pushState({}, '', '/');
    vi.mocked(authService.getStatus).mockRejectedValue(new Error('status unavailable'));
    mockAuth({ isInitialized: true, isAuthenticated: false, isLoading: false, user: null });

    render(<App />);

    expect(await screen.findByText('Login page')).toBeInTheDocument();
  });

  it.each([
    ['/dashboard', 'Overview page'],
    ['/flows', 'Overview page'],
    ['/flows/versions', 'Recovery page'],
    ['/flows/flow-42', 'Overview page'],
    ['/files', 'Overview page'],
    ['/updates', 'Updates page'],
    ['/libraries', 'Libraries page'],
  ])('redirects legacy route %s intentionally', async (path, expectedPage) => {
    window.history.pushState({}, '', path);
    mockAuth({
      isAuthenticated: true,
      isInitialized: true,
      isLoading: false,
      user: { id: 'admin', username: 'admin', role: 'admin', createdAt: '2024-01-01T00:00:00Z' },
    });

    render(<App />);

    expect(await screen.findByText(expectedPage)).toBeInTheDocument();
  });

  it.each(['/maintenance/updates', '/maintenance/libraries'])('denies non-admin direct access to %s', async (path) => {
    window.history.pushState({}, '', path);
    mockAuth({
      isAuthenticated: true,
      isInitialized: true,
      isLoading: false,
      user: { id: 'viewer', username: 'viewer', role: 'viewer', createdAt: '2024-01-01T00:00:00Z' },
    });

    render(<App />);

    await waitFor(() => expect(screen.getByText('Overview page')).toBeInTheDocument());
  });
});
