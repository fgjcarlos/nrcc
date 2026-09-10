import { describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { DashboardAccess } from './DashboardAccess';
import { server } from '@/test/msw/server';

const ok = (data: unknown) => HttpResponse.json({ success: true, data });
const discovery = { packages: { state: 'available' }, flows: { state: 'available' }, flowFuse: [{}], uiBases: [{ nodeId: 'one', path: '/dashboard', state: 'available' }, { nodeId: 'two', path: '/ops', state: 'available' }] };

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
    for (const message of [/changed elsewhere.*retry/i, /transaction is in progress.*retry/i, /readiness check is rolled back.*retry/i]) { await user.click(screen.getByRole('button', { name: 'Apply dashboard access' })); await screen.findByText(message); }
    expect(screen.queryByText('not-in-status')).not.toBeInTheDocument();
  });
});
