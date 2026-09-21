import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import App from './App';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { buildAuthMock } from '@/features/auth/__test-utils__/authMock';

vi.mock('@/features/auth/hooks/useAuth', () => ({ useAuth: vi.fn() }));
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
  beforeEach(() => vi.clearAllMocks());

  it.each([
    ['uninitialized', false, false, 'Setup page'],
    ['unauthenticated', true, false, 'Login page'],
    ['authenticated', true, true, 'Overview page'],
  ])('routes / deterministically for %s users', async (_state, isInitialized, isAuthenticated, expectedPage) => {
    window.history.pushState({}, '', '/');
    mockAuth({ isInitialized, isAuthenticated, isLoading: false, user: isAuthenticated ? undefined : null });

    render(<App />);

    expect(await screen.findByText(expectedPage)).toBeInTheDocument();
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
