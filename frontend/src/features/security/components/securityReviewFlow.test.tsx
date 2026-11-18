import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { toast } from 'sonner';
import { I18nProvider } from '@/i18n';
import { server } from '@/test/msw/server';
import { AdminAuthBoundaryCard } from './AdminAuthBoundaryCard';
import { HttpBasicAuthBoundaryCard } from './HttpBasicAuthBoundaryCard';

// Values typed into the form in tests; not credentials.
const TYPED = ['typed', 'in', 'test'].join('-');
const STORED_HASH = '$2b$10$storedhashstoredhashstoredhash';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

function capture(response: () => Response = () => ok({ configuration: {}, document: {} })) {
  const bodies: unknown[] = [];
  server.use(
    http.post('/api/config/apply', async ({ request }) => {
      bodies.push(await request.json());
      return response();
    }),
  );
  return bodies;
}

function wrap(node: React.ReactNode) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider>{node}</I18nProvider>
    </QueryClientProvider>,
  );
}

function renderAdmin(users: Array<{ username: string; permissions: '*' | 'read'; password?: string }>, raw = '') {
  const onApplied = vi.fn();
  wrap(
    <AdminAuthBoundaryCard
      // The admin API returns stored hashes; the type does not declare them.
      config={{ type: 'credentials', sessionExpiryTime: 3600, users } as never}
      rawSettingsContent={raw}
      expectedRevision="rev"
      editable
      onApplied={onApplied}
    />,
  );
  return onApplied;
}

const drawer = () => screen.getByTestId('review-changes-panel');
const live = (surface: string) => within(screen.getByTestId(surface)).getByTestId('change-circuit-live');

describe('Security review drawer + change circuit (P1 #1)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('Save opens the review drawer with redacted values and sends nothing', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    renderAdmin([{ username: 'admin', permissions: '*', password: STORED_HASH }]);

    await user.click(screen.getByTestId('boundary-admin-auth-add'));
    await user.type(screen.getAllByTestId('boundary-admin-auth-username')[1], 'alice');
    await user.type(screen.getAllByTestId('boundary-admin-auth-password')[1], TYPED);
    await user.type(screen.getAllByTestId('boundary-admin-auth-password')[0], TYPED);
    await user.click(screen.getByTestId('boundary-admin-auth-save'));

    const panel = drawer();
    expect(within(panel).getByRole('heading', { name: 'Review changes' })).toBeInTheDocument();
    expect(panel).toHaveTextContent('adminAuth.users[alice]');
    expect(panel).toHaveTextContent('adminAuth.users[admin].password');
    expect(panel).toHaveTextContent('[redacted]');
    expect(panel).not.toHaveTextContent(TYPED);
    expect(panel).not.toHaveTextContent('$2');
    expect(within(panel).getByTestId('review-changes-panel-group-hard')).toBeInTheDocument();
    expect(within(panel).getByTestId('review-changes-panel-consequence')).toHaveTextContent('2 changes');
    expect(bodies).toHaveLength(0);
  });

  it('Cancel closes the drawer and sends nothing', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    renderAdmin([{ username: 'admin', permissions: '*' }]);
    await user.selectOptions(screen.getByTestId('boundary-admin-auth-permission'), 'read');
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await user.click(within(drawer()).getAllByRole('button', { name: 'Cancel' }).slice(-1)[0]);
    expect(screen.queryByTestId('review-changes-panel')).toBeNull();
    expect(bodies).toHaveLength(0);
    // Reviewing closes the Diff contact (K2); nothing ran on the server.
    const steps = within(screen.getByTestId('boundary-admin-auth')).getByTestId('change-circuit-steps').children;
    expect([...steps].map((step) => step.getAttribute('data-state'))).toEqual([
      'passed', 'passed', 'idle', 'idle', 'idle', 'idle',
    ]);
  });

  it('Apply sends the existing payload and the circuit carries the outcome, with no toast', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    const onApplied = renderAdmin([{ username: 'admin', permissions: '*' }]);
    await user.selectOptions(screen.getByTestId('boundary-admin-auth-permission'), 'read');
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await user.click(within(drawer()).getByRole('button', { name: 'Apply and restart Node-RED' }));

    await waitFor(() => expect(onApplied).toHaveBeenCalled());
    expect(bodies).toEqual([
      {
        adminAuth: {
          type: 'credentials',
          users: [{ username: 'admin', permissions: 'read', password: '' }],
          sessionExpiryTime: 3600,
        },
        expectedRevision: 'rev',
      },
    ]);
    expect(screen.queryByTestId('review-changes-panel')).toBeNull();
    expect(live('boundary-admin-auth')).toHaveTextContent('Ready: Closed. Readiness OK');
    expect(toast.success).not.toHaveBeenCalled();
    expect(toast.error).not.toHaveBeenCalled();
  });

  it('a refused apply shows on the circuit and inline, with no toast', async () => {
    const user = userEvent.setup();
    capture(() =>
      HttpResponse.json(
        { success: false, error: { code: 'SETTINGS_REVISION_CONFLICT', message: 'x' }, timestamp: '' },
        { status: 409 },
      ),
    );
    renderAdmin([{ username: 'admin', permissions: '*' }]);
    await user.selectOptions(screen.getByTestId('boundary-admin-auth-permission'), 'read');
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await user.click(within(drawer()).getByTestId('review-changes-panel-apply'));

    await waitFor(() =>
      expect(live('boundary-admin-auth')).toHaveTextContent('Validate: Failed. Rejected; nothing written'),
    );
    expect(screen.getByTestId('boundary-admin-auth-error')).toHaveTextContent('Settings changed elsewhere');
    expect(toast.success).not.toHaveBeenCalled();
    expect(toast.error).not.toHaveBeenCalled();
  });

  it('warns in the drawer that enabling adminAuth puts the editor behind this login', async () => {
    const user = userEvent.setup();
    renderAdmin([]);
    await user.click(screen.getByTestId('boundary-admin-auth-add'));
    await user.type(screen.getByTestId('boundary-admin-auth-username'), 'alice');
    await user.click(screen.getByTestId('boundary-admin-auth-save'));

    const notice = within(drawer()).getByTestId('review-changes-panel-notice');
    expect(notice).toHaveTextContent('The Node-RED editor will require this login');
    expect(notice).toHaveTextContent('alice');
    expect(within(drawer()).getByTestId('review-changes-panel-apply')).toHaveAccessibleDescription(
      expect.stringContaining('The Node-RED editor will require this login'),
    );
  });

  it('does not warn when adminAuth is already on', async () => {
    const user = userEvent.setup();
    renderAdmin([{ username: 'admin', permissions: '*' }]);
    await user.selectOptions(screen.getByTestId('boundary-admin-auth-permission'), 'read');
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    expect(within(drawer()).queryByTestId('review-changes-panel-notice')).toBeNull();
  });

  it('runs the migration acknowledgement after the drawer, then applies', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    renderAdmin(
      [{ username: 'admin', permissions: '*' }],
      'module.exports = { nodeHttpAuth: { user: "n", pass: "[redacted]" } };',
    );
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    expect(drawer()).toHaveTextContent('nodeHttpAuth');
    await user.click(within(drawer()).getByTestId('review-changes-panel-apply'));

    const dialog = await screen.findByRole('dialog', { name: 'Apply authentication migration' });
    expect(bodies).toHaveLength(0);
    await user.click(within(dialog).getByTestId('confirmation-dialog-ack'));
    await user.click(within(dialog).getByRole('button', { name: /^confirm$/i }));
    await waitFor(() => expect(bodies).toHaveLength(1));
  });
});

describe('HttpBasicAuthBoundaryCard review drawer', () => {
  beforeEach(() => vi.clearAllMocks());

  it('redacts the stored hash and the typed password, then applies the same payload', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    const onApplied = vi.fn();
    wrap(
      <HttpBasicAuthBoundaryCard
        surface="httpNodeAuth"
        config={{ user: 'nodes', pass: STORED_HASH }}
        rawSettingsContent=""
        expectedRevision="rev"
        editable
        onApplied={onApplied}
      />,
    );
    await user.type(screen.getByTestId('boundary-httpNodeAuth-pass'), TYPED);
    await user.click(screen.getByTestId('boundary-httpNodeAuth-save'));

    const panel = drawer();
    expect(panel).toHaveTextContent('httpNodeAuth.pass');
    expect(panel).toHaveTextContent('[redacted]');
    expect(panel).not.toHaveTextContent(TYPED);
    expect(panel).not.toHaveTextContent('$2');
    expect(bodies).toHaveLength(0);

    await user.click(within(panel).getByTestId('review-changes-panel-apply'));
    await waitFor(() => expect(onApplied).toHaveBeenCalled());
    expect(bodies).toEqual([{ httpNodeAuth: { user: 'nodes', pass: TYPED }, expectedRevision: 'rev' }]);
    expect(live('boundary-httpNodeAuth')).toHaveTextContent('Ready: Closed');
    expect(toast.success).not.toHaveBeenCalled();
  });
});
