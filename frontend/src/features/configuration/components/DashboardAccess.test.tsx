import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { toast } from 'sonner';
import { i18n } from '@/i18n';
import enSecurity from '@/locales/en/security.json';
import esSecurity from '@/locales/es/security.json';
import { DashboardAccess } from './DashboardAccess';
import { server } from '@/test/msw/server';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) => HttpResponse.json({ success: true, data });
const discovery = { packages: { state: 'available' }, flows: { state: 'available' }, flowFuse: [{}], uiBases: [{ nodeId: 'one', path: '/dashboard', state: 'available' }, { nodeId: 'two', path: '/ops', state: 'available' }] };

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('DashboardAccess', () => {
  it('uses isolated, redacted policy payloads over the real transport', async () => {
    let payload: unknown;
    server.use(http.get('/api/dashboards/discovery', () => ok(discovery)), http.post('/api/dashboards/access', async ({ request }) => { payload = await request.json(); return ok({ document: {} }); }));
    const user = userEvent.setup();
    render(<DashboardAccess editable expectedRevision="revision" onApplied={() => undefined} />);
    await screen.findByText(/FlowFuse paths \/dashboard, \/ops/);
    await user.selectOptions(screen.getByLabelText('Dashboard target'), 'flowfuse');
    await user.type(screen.getByLabelText('Username'), 'operator');
    await user.type(screen.getByLabelText('Access secret'), 'do-not-render');
    await user.click(screen.getByRole('button', { name: 'Apply dashboard access' }));
    await waitFor(() => expect(payload).toEqual({ target: 'flowfuse', recipe: 'basic-auth', username: 'operator', secret: 'do-not-render', expectedRevision: 'revision' }));
    expect(screen.getByLabelText('Access secret')).toHaveValue('');
    expect(screen.getByRole('complementary')).toHaveTextContent(/never shown/);
  });

  it('sends no username for the reverse proxy recipe', async () => {
    let payload: unknown;
    server.use(http.get('/api/dashboards/discovery', () => ok({ ...discovery, legacy: { path: '/ui' } })), http.post('/api/dashboards/access', async ({ request }) => { payload = await request.json(); return ok({ document: {} }); }));
    const user = userEvent.setup();
    render(<DashboardAccess editable expectedRevision="revision" onApplied={() => undefined} />);
    await screen.findByText(/legacy \/ui/);
    await user.selectOptions(screen.getByLabelText('Access recipe'), 'reverse-proxy-sso');
    await user.type(screen.getByLabelText('Access secret'), 'proxy-secret');
    await user.click(screen.getByRole('button', { name: 'Apply dashboard access' }));
    await waitFor(() => expect(payload).toStrictEqual({ target: 'legacy', recipe: 'reverse-proxy-sso', secret: 'proxy-secret', expectedRevision: 'revision' }));
  });

  it('refuses unsafe discovery and explains stale transactions', async () => {
    server.use(http.get('/api/dashboards/discovery', () => ok({ ...discovery, flows: { state: 'malformed' } })));
    render(<DashboardAccess editable expectedRevision="revision" onApplied={() => undefined} />);
    expect(await screen.findByRole('button', { name: 'Apply dashboard access' })).toBeDisabled();
    expect(screen.getByText(/flows malformed/)).toBeVisible();
  });

  it('guides stale, retry, and rollback outcomes without exposing secrets', async () => {
    let calls = 0;
    server.use(http.get('/api/dashboards/discovery', () => ok({ ...discovery, legacy: { path: '/ui' } })), http.post('/api/dashboards/access', () => HttpResponse.json({ success: false, error: { code: ++calls === 1 ? 'SETTINGS_REVISION_CONFLICT' : calls === 2 ? 'APPLY_IN_FLIGHT' : 'DASHBOARD_POLICY_APPLY_FAILED' } }, { status: calls === 1 ? 409 : 500 })));
    const user = userEvent.setup();
    render(<DashboardAccess editable expectedRevision="revision" onApplied={() => undefined} />);
    await screen.findByText(/legacy \/ui/); await user.type(screen.getByLabelText('Username'), 'operator'); await user.type(screen.getByLabelText('Access secret'), 'not-in-status');
    for (const message of [/changed elsewhere.*retry/i, /transaction is in progress.*retry/i, /readiness check is rolled back.*retry/i]) { await user.click(screen.getByRole('button', { name: 'Apply dashboard access' })); await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(message)); }
    expect(screen.queryByText('not-in-status')).not.toBeInTheDocument();
    expect(toast.error).not.toHaveBeenCalled();
  });

  it('reports a successful apply inline, not as a toast', async () => {
    server.use(http.get('/api/dashboards/discovery', () => ok({ ...discovery, legacy: { path: '/ui' } })), http.post('/api/dashboards/access', () => ok({ document: {} })));
    const onApplied = vi.fn();
    const user = userEvent.setup();
    render(<DashboardAccess editable expectedRevision="revision" onApplied={onApplied} />);
    await screen.findByText(/legacy \/ui/);
    await user.type(screen.getByLabelText('Username'), 'operator');
    await user.type(screen.getByLabelText('Access secret'), 'secret');
    await user.click(screen.getByRole('button', { name: 'Apply dashboard access' }));
    await waitFor(() => expect(screen.getAllByRole('status').some((node) => node.textContent === enSecurity.dashboardAccess.applied)).toBe(true));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(onApplied).toHaveBeenCalledTimes(1);
    expect(toast.success).not.toHaveBeenCalled();
  });

  it('shows a missing legacy dashboard as a disabled option with helper text', async () => {
    server.use(http.get('/api/dashboards/discovery', () => ok(discovery)));
    render(<DashboardAccess editable expectedRevision="revision" onApplied={() => undefined} />);
    await screen.findByText(/FlowFuse paths \/dashboard, \/ops/);
    const legacy = screen.getByRole('option', { name: 'Legacy dashboard (deprecated)' });
    expect(legacy).toBeDisabled();
    for (const option of screen.getAllByRole('option')) expect(option.textContent).not.toMatch(/unavailable/i);
    expect(screen.getByLabelText('Dashboard target')).toHaveAccessibleDescription(enSecurity.dashboardAccess.legacyUnavailable);
  });

  it('carries no heading of its own: the Security row titles it', async () => {
    server.use(http.get('/api/dashboards/discovery', () => ok(discovery)));
    render(<DashboardAccess editable expectedRevision="revision" onApplied={() => undefined} />);
    await screen.findByText(/FlowFuse paths/);
    expect(screen.queryByRole('heading', { level: 2 })).not.toBeInTheDocument();
  });

  it('renders no English copy in Spanish, from loading to apply', async () => {
    const flatten = (value: unknown): string[] => (typeof value === 'string' ? [value] : Object.values(value as object).flatMap(flatten));
    const es = flatten([esSecurity.dashboardAccess, esSecurity.applyErrors]);
    const english = flatten([enSecurity.dashboardAccess, enSecurity.applyErrors])
      .flatMap((value) => value.split(/\{\{[^}]+\}\}/))
      .map((part) => part.trim())
      .filter((part) => part.length > 3 && !es.some((value) => value.includes(part)))
      .concat(['Loading dashboard discovery', 'path unavailable', 'none discovered', '(deprecated)', 'Dashboard target', 'Access recipe', 'unavailable']);
    const seen: string[] = [];
    const snapshot = () => seen.push(document.body.textContent ?? '', ...[...document.querySelectorAll('[aria-label]')].map((node) => node.getAttribute('aria-label') ?? ''));

    let calls = 0;
    server.use(
      http.get('/api/dashboards/discovery', () => ok({ ...discovery, legacy: { path: '/ui' }, uiBases: [{ nodeId: 'one', state: 'available' }] })),
      http.post('/api/dashboards/access', () => (++calls === 1 ? HttpResponse.json({ success: false, error: { code: 'DASHBOARD_POLICY_APPLY_FAILED' } }, { status: 500 }) : ok({ document: {} }))),
    );
    await i18n.changeLanguage('es');
    const user = userEvent.setup();
    const { unmount } = render(<DashboardAccess editable expectedRevision="revision" onApplied={() => undefined} />);
    expect(screen.getByRole('status')).toHaveTextContent(esSecurity.dashboardAccess.loading);
    snapshot();
    await screen.findByText(/legacy \/ui \(deprecado\)/);
    expect(screen.getByText(/one \(ruta no disponible\)/)).toBeInTheDocument();
    snapshot();
    await user.type(screen.getByLabelText(esSecurity.dashboardAccess.username), 'operador');
    await user.type(screen.getByLabelText(esSecurity.dashboardAccess.secret), 'secreto');
    await user.click(screen.getByRole('button', { name: esSecurity.dashboardAccess.apply }));
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(esSecurity.applyErrors.generic));
    snapshot();
    await user.click(screen.getByRole('button', { name: esSecurity.dashboardAccess.apply }));
    await screen.findByText(esSecurity.dashboardAccess.applied);
    snapshot();
    unmount();

    server.use(http.get('/api/dashboards/discovery', () => HttpResponse.json({ success: false }, { status: 500 })));
    render(<DashboardAccess editable expectedRevision="revision" onApplied={() => undefined} />);
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(esSecurity.dashboardAccess.discoveryFailed));
    expect(screen.getByText(new RegExp(esSecurity.dashboardAccess.noneDiscovered))).toBeInTheDocument();
    snapshot();

    expect(english.filter((part) => seen.some((text) => text.includes(part)))).toEqual([]);
  });
});
