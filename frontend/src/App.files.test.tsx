import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import App from './App';
import { useAuth } from '@/features/auth/hooks/useAuth';
import { buildAuthMock } from '@/features/auth/__test-utils__/authMock';

vi.mock('@/features/auth/hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

vi.mock('@/features/auth/components/LoginView', () => ({
  LoginView: () => <div>Login page</div>,
}));

vi.mock('@/features/files/components/FilesView', () => ({
  FilesView: () => <div>Files page</div>,
}));

vi.mock('@/shared/components', async () => {
  const actual = await vi.importActual<typeof import('@/shared/components')>('@/shared/components');
  return {
    ...actual,
    ThemeToggle: () => null,
  };
});

vi.mock('@/features/updates/components/UpdateNotificationChip', () => ({
  UpdateNotificationChip: () => null,
}));

vi.mock('@/shared/components/command-palette', () => ({
  CommandPalette: () => null,
}));

describe('files route protection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.history.pushState({}, '', '/files');
  });

  it('redirects unauthenticated users from /files to login', async () => {
    vi.mocked(useAuth).mockReturnValue(
      buildAuthMock({
        isAuthenticated: false,
        isInitialized: true,
        isLoading: false,
        user: null,
      }),
    );

    render(<App />);

    await waitFor(() => expect(screen.getByText('Login page')).toBeInTheDocument());
    expect(screen.queryByText('Files page')).not.toBeInTheDocument();
  });

  it('allows authenticated users to open /files', async () => {
    vi.mocked(useAuth).mockReturnValue(
      buildAuthMock({
        isAuthenticated: true,
        isInitialized: true,
        isLoading: false,
      }),
    );

    render(<App />);

    await waitFor(() => expect(screen.getByText('Files page')).toBeInTheDocument());
  });
});
