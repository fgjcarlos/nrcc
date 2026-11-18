import { useState } from 'react';
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { I18nProvider } from '@/i18n';
import { server } from '@/test/msw/server';
import { AdminAuthBoundaryCard } from './AdminAuthBoundaryCard';
import { HttpBasicAuthBoundaryCard, type HttpBasicAuthSurface } from './HttpBasicAuthBoundaryCard';

// Regression guards for the Security review drawer + change circuit (L42, 614aa4b).

// Values typed into the form in tests; not credentials.
const TYPED = ['typed', 'in', 'test'].join('-');
const STORED_HASH = '$2b$10$storedhashstoredhashstoredhash';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

function capture(response: () => Response | Promise<Response> = () => ok({ configuration: {}, document: {} })) {
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

type User = { username: string; permissions: '*' | 'read'; password?: string };

function renderAdmin(users: User[], { expectedRevision }: { expectedRevision?: string } = { expectedRevision: 'rev' }) {
  const onApplied = vi.fn();
  wrap(
    <AdminAuthBoundaryCard
      // The admin API returns stored hashes; the type does not declare them.
      config={{ type: 'credentials', sessionExpiryTime: 3600, users } as never}
      rawSettingsContent=""
      expectedRevision={expectedRevision}
      editable
      onApplied={onApplied}
    />,
  );
  return onApplied;
}

const drawer = () => screen.getByTestId('review-changes-panel');
const queryDrawer = () => screen.queryByTestId('review-changes-panel');
const applyButton = () => within(drawer()).getByTestId('review-changes-panel-apply');
/** K1–K6 contacts of the admin surface circuit. */
const contacts = () => [
  ...within(screen.getByTestId('boundary-admin-auth')).getByTestId('change-circuit-steps').children,
];

/** One edit (permission) and Save: the drawer is open with one pending change. */
async function openReview(user: ReturnType<typeof userEvent.setup>) {
  await user.selectOptions(screen.getByTestId('boundary-admin-auth-permission'), 'read');
  await user.click(screen.getByTestId('boundary-admin-auth-save'));
  expect(drawer()).toBeInTheDocument();
}

describe('Security review drawer guards', () => {
  beforeEach(() => vi.clearAllMocks());

  it('Esc closes the open drawer and sends nothing', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    renderAdmin([{ username: 'admin', permissions: '*' }]);
    await openReview(user);

    await user.keyboard('{Escape}');

    expect(queryDrawer()).toBeNull();
    expect(bodies).toHaveLength(0);
  });

  it('two Apply clicks in the same tick send exactly one request', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    const onApplied = renderAdmin([{ username: 'admin', permissions: '*' }]);
    await openReview(user);

    const apply = within(drawer()).getByRole('button', { name: 'Apply and restart Node-RED' });
    // One act: React has not re-rendered the disabled button between the clicks.
    act(() => {
      apply.click();
      apply.click();
    });

    await waitFor(() => expect(onApplied).toHaveBeenCalled());
    expect(bodies).toHaveLength(1);
    expect(onApplied).toHaveBeenCalledTimes(1);
  });

  it('locks the drawer while the request is pending', async () => {
    const user = userEvent.setup();
    let release!: () => void;
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    const bodies = capture(async () => {
      await held;
      return ok({ configuration: {}, document: {} });
    });
    const onApplied = renderAdmin([{ username: 'admin', permissions: '*' }]);
    await openReview(user);
    await user.click(applyButton());
    await waitFor(() => expect(bodies).toHaveLength(1));

    const buttons = within(drawer()).getAllByRole('button');
    expect(buttons.length).toBeGreaterThanOrEqual(3);
    for (const button of buttons) expect(button).toBeDisabled();

    await user.keyboard('{Escape}');
    expect(queryDrawer()).toBeInTheDocument();

    const backdrop = screen.getByTestId('review-changes-panel-portal').firstElementChild as HTMLElement;
    fireEvent.click(backdrop);
    expect(queryDrawer()).toBeInTheDocument();

    release();
    await waitFor(() => expect(onApplied).toHaveBeenCalled());
    expect(queryDrawer()).toBeNull();
  });

  it('a 500 APPLY_ERROR leaves K3–K6 not confirmed and draws no rollback', async () => {
    const user = userEvent.setup();
    capture(() =>
      HttpResponse.json(
        { success: false, error: { code: 'APPLY_ERROR', message: 'x' }, timestamp: '' },
        { status: 500 },
      ),
    );
    renderAdmin([{ username: 'admin', permissions: '*' }]);
    await openReview(user);
    await user.click(applyButton());

    await waitFor(() => expect(contacts()[2]).toHaveAttribute('data-state', 'unknown'));
    const serverContacts = contacts().slice(2);
    expect(serverContacts.map((contact) => contact.getAttribute('data-state'))).toEqual([
      'unknown', 'unknown', 'unknown', 'unknown',
    ]);
    serverContacts.forEach((contact, i) => {
      expect(contact).toHaveTextContent(`K${i + 3}`);
      expect(contact).toHaveTextContent('Not confirmed');
    });
    expect(screen.queryByTestId('change-circuit-rollback')).toBeNull();
  });

  it('without a settings revision Apply sends nothing, shows the inline error and keeps K3–K6 waiting', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    const onApplied = renderAdmin([{ username: 'admin', permissions: '*' }], {});
    await openReview(user);
    await user.click(applyButton());

    expect(await screen.findByTestId('boundary-admin-auth-error')).toHaveTextContent(
      'The settings revision is unavailable. Refresh before retrying.',
    );
    expect(bodies).toHaveLength(0);
    expect(onApplied).not.toHaveBeenCalled();
    const serverContacts = contacts().slice(2);
    expect(serverContacts.map((contact) => contact.getAttribute('data-state'))).toEqual([
      'idle', 'idle', 'idle', 'idle',
    ]);
    serverContacts.forEach((contact) => expect(contact).toHaveTextContent('Waiting'));
  });

  it('removing every admin user does not show the lock-out notice', async () => {
    const user = userEvent.setup();
    renderAdmin([{ username: 'admin', permissions: '*' }]);
    await user.click(screen.getByTestId('boundary-admin-auth-remove'));
    await user.click(screen.getByTestId('boundary-admin-auth-save'));

    expect(drawer()).toHaveTextContent('adminAuth');
    expect(within(drawer()).queryByTestId('review-changes-panel-notice')).toBeNull();
    expect(drawer()).not.toHaveTextContent('The Node-RED editor will require this login');
  });
});

describe('No secret in the DOM after Apply', () => {
  beforeEach(() => vi.clearAllMocks());

  it('admin user added with a password', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    const onApplied = renderAdmin([{ username: 'admin', permissions: '*', password: STORED_HASH }]);
    await user.click(screen.getByTestId('boundary-admin-auth-add'));
    await user.type(screen.getAllByTestId('boundary-admin-auth-username')[1], 'alice');
    await user.type(screen.getAllByTestId('boundary-admin-auth-password')[1], TYPED);
    await user.click(screen.getByTestId('boundary-admin-auth-save'));
    await user.click(applyButton());

    await waitFor(() => expect(onApplied).toHaveBeenCalled());
    expect(bodies).toHaveLength(1);
    expect(document.body.innerHTML).not.toContain(TYPED);
    expect(document.body.innerHTML).not.toContain('$2');
  });

  it('new HTTP basic-auth surface with a typed password', async () => {
    const user = userEvent.setup();
    const bodies = capture();
    // Mirrors SecurityView: onApplied refetches and the surface receives the stored config.
    function Refetching() {
      const [config, setConfig] = useState<HttpBasicAuthSurface>(null);
      return (
        <HttpBasicAuthBoundaryCard
          surface="httpNodeAuth"
          config={config}
          rawSettingsContent=""
          expectedRevision="rev"
          editable
          onApplied={() => setConfig({ user: 'nodes', pass: STORED_HASH })}
        />
      );
    }
    wrap(<Refetching />);
    await user.type(screen.getByTestId('boundary-httpNodeAuth-user'), 'nodes');
    await user.type(screen.getByTestId('boundary-httpNodeAuth-pass'), TYPED);
    await user.click(screen.getByTestId('boundary-httpNodeAuth-save'));
    await user.click(applyButton());

    await waitFor(() => expect(screen.getByTestId('boundary-httpNodeAuth-user')).toHaveValue('nodes'));
    await waitFor(() => expect(screen.getByTestId('boundary-httpNodeAuth-pass')).toHaveValue(''));
    expect(bodies).toEqual([{ httpNodeAuth: { user: 'nodes', pass: TYPED }, expectedRevision: 'rev' }]);
    expect(document.body.innerHTML).not.toContain(TYPED);
    // The stored-hash hint prints the hash by design; it is the only place allowed to.
    expect(screen.getByTestId('boundary-httpNodeAuth-stored-hash')).toHaveTextContent(STORED_HASH);
    const rest = document.body.cloneNode(true) as HTMLElement;
    rest.querySelector('[data-testid="boundary-httpNodeAuth-stored-hash"]')?.remove();
    expect(rest.innerHTML).not.toContain('$2');
  });
});

describe('ChangeCircuit button hiding on a Security surface', () => {
  it('[&_button]:hidden reaches only the circuit, which holds exactly one button', () => {
    renderAdmin([{ username: 'admin', permissions: '*' }]);
    const surface = screen.getByTestId('boundary-admin-auth');
    const circuit = within(surface).getByTestId('change-circuit');

    expect(circuit).toHaveClass('[&_button]:hidden');
    expect(circuit.querySelectorAll('button')).toHaveLength(1);
    for (const id of ['boundary-admin-auth-save', 'boundary-admin-auth-add']) {
      const button = within(surface).getByTestId(id);
      expect(circuit.contains(button)).toBe(false);
      expect(button.closest('.\\[\\&_button\\]\\:hidden')).toBeNull();
    }
  });
});
