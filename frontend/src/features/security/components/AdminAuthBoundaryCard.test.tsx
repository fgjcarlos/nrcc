import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { I18nProvider } from '@/i18n';
import { server } from '@/test/msw/server';
import { AdminAuthBoundaryCard } from './AdminAuthBoundaryCard';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) =>
  HttpResponse.json({
    success: true,
    data,
    timestamp: new Date(0).toISOString(),
  });

function buildConfig(
  overrides: {
    users?: Array<{ username?: string; permissions?: '*' | 'read' }>;
    sessionExpiryTime?: number;
    type?: 'credentials';
  } = {},
): Parameters<typeof AdminAuthBoundaryCard>[0]['config'] {
  return {
    type: 'credentials',
    sessionExpiryTime: 3600,
    users: [{ username: 'admin', permissions: '*' }],
    ...overrides,
  };
}

function renderBoundary(props: Parameters<typeof AdminAuthBoundaryCard>[0]) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <AdminAuthBoundaryCard {...props} />
      </I18nProvider>
    </QueryClientProvider>,
  );
}

describe('AdminAuthBoundaryCard', () => {
  it('renders the success palette when the surface is configured', () => {
    renderBoundary({
      config: buildConfig(),
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
      onApplied: vi.fn(),
    });
    const card = screen.getByTestId('boundary-admin-auth');
    expect(card).toBeInTheDocument();
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-success/);
  });

  it('renders the warning palette when no users are configured', () => {
    renderBoundary({
      config: buildConfig({ users: [] }),
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
      onApplied: vi.fn(),
    });
    const card = screen.getByTestId('boundary-admin-auth');
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-warning/);
  });

  it('disables the fieldset when not editable', () => {
    renderBoundary({
      config: buildConfig(),
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: false,
      onApplied: vi.fn(),
    });
    const fieldset = screen.getByTestId('boundary-admin-auth-fieldset');
    expect(fieldset).toBeDisabled();
    expect(screen.getByTestId('boundary-admin-auth-readonly')).toBeInTheDocument();
  });

  it('adds a user row when the add button is clicked', async () => {
    const user = userEvent.setup();
    renderBoundary({
      config: buildConfig({ users: [] }),
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
      onApplied: vi.fn(),
    });
    expect(screen.queryAllByTestId('boundary-admin-auth-user-row')).toHaveLength(0);
    await user.click(screen.getByTestId('boundary-admin-auth-add'));
    expect(screen.getAllByTestId('boundary-admin-auth-user-row')).toHaveLength(1);
  });

  it('removes a user row when the trash icon is clicked', async () => {
    const user = userEvent.setup();
    renderBoundary({
      config: buildConfig({ users: [{ username: 'admin', permissions: '*' }] }),
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
      onApplied: vi.fn(),
    });
    expect(screen.getAllByTestId('boundary-admin-auth-user-row')).toHaveLength(1);
    await user.click(screen.getByTestId('boundary-admin-auth-remove'));
    expect(screen.queryAllByTestId('boundary-admin-auth-user-row')).toHaveLength(0);
  });

  it('submits the adminAuth payload and calls onApplied on success', async () => {
    const user = userEvent.setup();
    const onApplied = vi.fn();
    let posted: unknown;
    server.use(
      http.post('/api/config/apply', async ({ request }) => {
        posted = await request.json();
        return ok({ configuration: {}, document: {} });
      }),
    );
    renderBoundary({
      config: buildConfig({ users: [{ username: 'admin', permissions: '*' }] }),
      rawSettingsContent: '',
      expectedRevision: 'current-rev',
      editable: true,
      onApplied,
    });
    // The save button stays disabled until the operator has touched the
    // form; mark the session expiry field dirty so click() fires apply().
    await user.clear(screen.getByTestId('boundary-admin-auth-expiry'));
    await user.type(screen.getByTestId('boundary-admin-auth-expiry'), '7200');
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await waitFor(() => expect(onApplied).toHaveBeenCalled());
    const payload = (posted as { adminAuth?: { type: string; users?: Array<{ username: string; password?: string }> } }).adminAuth;
    expect(payload?.type).toBe('credentials');
    expect(payload?.users?.[0]?.username).toBe('admin');
    // The password stays empty when the operator didn't retype it —
    // confirming slice E only re-sends the surface that was actually
    // edited.
    expect(payload?.users?.[0]?.password).toBe('');
  });

  it('surfaces an apply error when /api/config/apply returns a conflict', async () => {
    const user = userEvent.setup();
    server.use(
      http.post('/api/config/apply', () =>
        HttpResponse.json(
          {
            success: false,
            error: { code: 'SETTINGS_REVISION_CONFLICT', message: 'conflict' },
            timestamp: new Date(0).toISOString(),
          },
          { status: 409 },
        ),
      ),
    );
    renderBoundary({
      config: buildConfig({ users: [{ username: 'admin', permissions: '*' }] }),
      rawSettingsContent: '',
      expectedRevision: 'current-rev',
      editable: true,
      onApplied: vi.fn(),
    });
    // Mark the form dirty so save fires apply (the button stays
    // disabled while the operator hasn't touched anything).
    await user.clear(screen.getByTestId('boundary-admin-auth-expiry'));
    await user.type(screen.getByTestId('boundary-admin-auth-expiry'), '1800');
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await waitFor(() =>
      expect(screen.getByTestId('boundary-admin-auth-error')).toBeInTheDocument(),
    );
  });

  it('warns about legacy aliases and asks for explicit acknowledgement', async () => {
    const user = userEvent.setup();
    renderBoundary({
      config: buildConfig(),
      rawSettingsContent:
        'module.exports = { nodeHttpAuth: { user: "n", pass: "[redacted]" } };',
      expectedRevision: 'current-rev',
      editable: true,
      onApplied: vi.fn(),
    });
    expect(screen.getByTestId('boundary-admin-auth-legacy')).toBeInTheDocument();
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await waitFor(() =>
      expect(
        screen.getByRole('dialog'),
      ).toBeInTheDocument(),
    );
  });

  it('refuses to apply when the revision is missing', async () => {
    const user = userEvent.setup();
    renderBoundary({
      config: buildConfig({ users: [{ username: 'admin', permissions: '*' }] }),
      rawSettingsContent: '',
      // expectedRevision intentionally omitted
      editable: true,
      onApplied: vi.fn(),
    });
    // Same dirty-trick as the conflict-error test above.
    await user.clear(screen.getByTestId('boundary-admin-auth-expiry'));
    await user.type(screen.getByTestId('boundary-admin-auth-expiry'), '1800');
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await waitFor(() =>
      expect(screen.getByTestId('boundary-admin-auth-error')).toBeInTheDocument(),
    );
  });

  it('renders the empty-state hint when no users exist and the form is editable', () => {
    renderBoundary({
      config: buildConfig({ users: [] }),
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
      onApplied: vi.fn(),
    });
    expect(screen.getByTestId('boundary-admin-auth-empty')).toBeInTheDocument();
  });
});
