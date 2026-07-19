import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from './App';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { buildAuthMock } from '@/features/auth/__test-utils__/authMock';

vi.mock('@/features/auth/hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

// Mock every lazy-loaded view with a stable sentinel so we can assert
// which view rendered without exercising the real views' data flows.
vi.mock('@/features/auth/components/LoginView', () => ({
  LoginView: () => <div>Login page</div>,
}));

vi.mock('@/features/dashboard/components/DashboardView', () => ({
  DashboardView: () => <div>Dashboard page</div>,
}));

vi.mock('@/features/configuration/components/ConfigurationView', () => ({
  ConfigurationView: () => <div>Configuration page</div>,
}));

vi.mock('@/features/auth/components/ProfileView', () => ({
  ProfileView: () => <div>Profile page</div>,
}));

vi.mock('@/features/auth/components/UsersView', () => ({
  UsersView: () => <div>Users page</div>,
}));

vi.mock('@/features/updates/components/UpdatesView', () => ({
  UpdatesView: () => <div>Updates page</div>,
}));

vi.mock('@/features/libraries/components/LibrariesView', () => ({
  LibrariesView: () => <div>Libraries page</div>,
}));

vi.mock('@/features/flows/components/FlowsView', () => ({
  FlowsView: () => <div>Flows page</div>,
}));

vi.mock('@/features/flows/components/FlowVersionsView', () => ({
  FlowVersionsView: () => <div>Flow versions page</div>,
}));

vi.mock('@/features/flows/components/FlowDetailView', () => ({
  FlowDetailView: () => <div>Flow detail page</div>,
}));

vi.mock('@/features/bootstrap/components/BootstrapView', () => ({
  BootstrapView: () => <div>Bootstrap page</div>,
}));

vi.mock('@/features/env-vars/components/EnvVarsView', () => ({
  EnvVarsView: () => <div>Env vars page</div>,
}));

vi.mock('@/features/backups/components/BackupsView', () => ({
  BackupsView: () => <div>Backups page</div>,
}));

vi.mock('@/features/files/components/FilesView', () => ({
  FilesView: () => <div>Files page</div>,
}));

vi.mock('@/features/updates/components/UpdateNotificationChip', () => ({
  UpdateNotificationChip: () => null,
}));

vi.mock('@/shared/components/command-palette', () => ({
  CommandPalette: () => null,
}));

function mockAuthenticated(user?: { role?: 'admin' | 'viewer' }) {
  vi.mocked(useAuth).mockReturnValue(
    buildAuthMock({
      isAuthenticated: true,
      isInitialized: true,
      isLoading: false,
      user: {
        id: 'mock-user-id',
        username: 'mock-user',
        role: user?.role ?? 'viewer',
        createdAt: '2024-01-01T00:00:00.000Z',
      },
    }),
  );
}

describe('sidebar navigation (#365)', () => {
  let localStorageStub: Record<string, string>;

  beforeEach(() => {
    vi.clearAllMocks();
    // jsdom 29 (vitest 4.x default) does not seed `localStorage` in this
    // sandbox; Sidebar.tsx reads `localStorage.getItem` during its useState
    // initializer and would throw `Cannot read properties of undefined
    // (reading 'getItem')`. Stub a minimal in-memory store on the global
    // so the Sidebar can render. (The pre-existing failures on main also
    // stem from this gap in the test harness — see LayoutShell.test.tsx
    // and App.files.test.tsx.)
    localStorageStub = {};
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: {
        getItem: (key: string) =>
          Object.prototype.hasOwnProperty.call(localStorageStub, key)
            ? localStorageStub[key]
            : null,
        setItem: (key: string, value: string) => {
          localStorageStub[key] = String(value);
        },
        removeItem: (key: string) => {
          delete localStorageStub[key];
        },
        clear: () => {
          localStorageStub = {};
        },
        key: () => null,
        length: 0,
      },
    });
    window.history.pushState({}, '', '/dashboard');
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

  it('updates the main content area when the user clicks a sidebar link', async () => {
    mockAuthenticated();
    render(<App />);

    // Sanity: dashboard renders on first load.
    expect(await screen.findByText('Dashboard page')).toBeInTheDocument();

    // Click Configuration in the sidebar. The main content area should
    // render the Configuration view, NOT keep the dashboard view.
    await userEvent.click(screen.getByRole('link', { name: /Configuration/ }));

    await waitFor(() => {
      expect(screen.getByText('Configuration page')).toBeInTheDocument();
    });
    expect(screen.queryByText('Dashboard page')).not.toBeInTheDocument();
  });

  it('navigates Configuration -> Dashboard -> Configuration without showing a stale view', async () => {
    mockAuthenticated();
    window.history.pushState({}, '', '/configuration');
    render(<App />);

    expect(await screen.findByText('Configuration page')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('link', { name: /Dashboard/ }));
    await waitFor(() => {
      expect(screen.getByText('Dashboard page')).toBeInTheDocument();
    });
    expect(screen.queryByText('Configuration page')).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('link', { name: /Configuration/ }));
    await waitFor(() => {
      expect(screen.getByText('Configuration page')).toBeInTheDocument();
    });
    expect(screen.queryByText('Dashboard page')).not.toBeInTheDocument();
  });

  it('also routes through the admin Users link', async () => {
    mockAuthenticated({ role: 'admin' });
    window.history.pushState({}, '', '/dashboard');
    render(<App />);

    expect(await screen.findByText('Dashboard page')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('link', { name: /Users/ }));
    await waitFor(() => {
      expect(screen.getByText('Users page')).toBeInTheDocument();
    });
    expect(screen.queryByText('Dashboard page')).not.toBeInTheDocument();
  });
});
