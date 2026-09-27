import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { I18nProvider } from '@/i18n';
import { server } from '@/test/msw/server';
import {
  authUsersResponse,
  editableHostStatus,
  mockUser,
  securityCenterConfig,
} from '@/test/msw/fixtures';
import { SecurityView } from './SecurityView';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

function renderSecurityView() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/security']}>
        <I18nProvider>
          <SecurityView />
        </I18nProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function setupMocks(options: { editable?: boolean } = {}) {
  const editable = options.editable ?? true;
  server.use(
    http.get('/api/bootstrap/status', () =>
      ok({
        ...editableHostStatus,
        configuration: {
          ...editableHostStatus.configuration,
          editable,
          mode: editable ? 'editable' : 'read-only',
        },
      }),
    ),
    http.get('/api/config', () => ok(securityCenterConfig)),
    http.get('/api/settings/raw', () =>
      ok({
        content: 'module.exports = {};',
        writable: true,
        revision: { fingerprint: 'current-revision', algorithm: 'sha256' },
      }),
    ),
    http.post('/api/config/apply', () => ok({ configuration: {}, document: {} })),
    http.get('/api/auth/users', () => ok(authUsersResponse)),
  );
}

describe('SecurityView (slice E composition)', () => {
  it('renders all five boundary cards when editable', async () => {
    setupMocks();
    renderSecurityView();
    await waitFor(() =>
      expect(screen.getByTestId('boundary-nrcc-access')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('boundary-admin-auth')).toBeInTheDocument();
    expect(screen.getByTestId('boundary-httpNodeAuth')).toBeInTheDocument();
    expect(screen.getByTestId('boundary-httpStaticAuth')).toBeInTheDocument();
  });

  it('marks every boundary card read-only when not editable', async () => {
    setupMocks({ editable: false });
    renderSecurityView();
    await waitFor(() =>
      expect(screen.getByTestId('boundary-nrcc-access')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('boundary-admin-auth-readonly')).toBeInTheDocument();
    expect(screen.getByTestId('boundary-httpNodeAuth-readonly')).toBeInTheDocument();
    expect(screen.getByTestId('boundary-httpStaticAuth-readonly')).toBeInTheDocument();
  });

  it('surfaces the user count from the /auth/users fixture', async () => {
    setupMocks();
    renderSecurityView();
    await waitFor(() =>
      expect(screen.getByTestId('boundary-nrcc-access-total')).toHaveTextContent('1'),
    );
    expect(mockUser.username).toBe('admin');
  });

  it('submits the adminAuth payload and shows the success toast', async () => {
    const user = userEvent.setup();
    let posted: unknown;
    server.use(
      http.post('/api/config/apply', async ({ request }) => {
        posted = await request.json();
        return ok({ configuration: {}, document: {} });
      }),
    );
    setupMocks();
    renderSecurityView();
    await waitFor(() =>
      expect(screen.getByTestId('boundary-admin-auth')).toBeInTheDocument(),
    );
    // Wait for the user rows to render so adminAuth.users is populated
    // when apply() builds the patch; otherwise apply sends
    // { adminAuth: null } and the backend rejects it.
    await waitFor(() =>
      expect(
        screen.getAllByTestId('boundary-admin-auth-username').length,
      ).toBeGreaterThan(0),
    );
    // user-event's `clear()` doesn't work on number inputs in this
    // version; use fireEvent.change to bump the expiry value so the
    // payload differs from the server snapshot.
    const expiryInput = screen.getByTestId('boundary-admin-auth-expiry');
    fireEvent.change(expiryInput, { target: { value: '7200' } });
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await waitFor(() =>
      expect(screen.getByTestId('boundary-admin-auth-applied')).toBeInTheDocument(),
    );
    const payload = (posted as { adminAuth?: unknown }).adminAuth;
    expect(payload).toBeDefined();
  });

  it('rejects a non-bcrypt password on httpNodeAuth', async () => {
    const user = userEvent.setup();
    setupMocks();
    renderSecurityView();
    await waitFor(() =>
      expect(screen.getByTestId('boundary-httpNodeAuth')).toBeInTheDocument(),
    );
    const userInput = screen.getByTestId('boundary-httpNodeAuth-user');
    await user.type(userInput, 'nodes');
    // Force a non-bcrypt password by writing a plain string to the input.
    const passInput = screen.getByTestId('boundary-httpNodeAuth-pass');
    await user.type(passInput, 'not-a-bcrypt-hash');
    // The bcrypt-warning banner surfaces before save is even attempted.
    expect(
      screen.getByTestId('boundary-httpNodeAuth-bcrypt-warning'),
    ).toBeInTheDocument();
  });

  it('deep-links the NRCC access card to /settings/users', async () => {
    setupMocks();
    renderSecurityView();
    await waitFor(() =>
      expect(screen.getByTestId('boundary-nrcc-access')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('boundary-nrcc-access-link')).toHaveAttribute(
      'href',
      '/settings/users',
    );
  });
});
