import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';
import { ConfigurationView } from './ConfigurationView';
import { server } from '@/test/msw/server';
import { editableHostStatus, securityCenterConfig } from '@/test/msw/fixtures';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) => HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

function renderView() {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><ConfigurationView /></QueryClientProvider>);
}

describe('Security Center', () => {
  let posted: unknown[];

  beforeEach(() => {
    posted = [];
    server.use(
      http.get('/api/bootstrap/status', () => ok(editableHostStatus)),
      http.get('/api/config', () => ok(securityCenterConfig)),
      http.get('/api/settings/raw', () => ok({ content: 'module.exports = { nodeHttpAuth: { user: "nodes", pass: "[redacted]" } };', writable: true, revision: { fingerprint: 'current-revision', algorithm: 'sha256' } })),
      http.post('/api/config/apply', async ({ request }) => {
        posted.push(await request.json());
        return ok({});
      }),
    );
  });

  it('uses canonical forms, preserves redaction, and requires migration confirmation', async () => {
    const user = userEvent.setup();
    renderView();
    await user.click(await screen.findByRole('button', { name: 'Authentication' }));

    expect(screen.getByRole('heading', { name: 'Security Center' })).toBeVisible();
    expect(screen.getByRole('complementary', { name: 'Redacted transaction preview' })).toHaveTextContent(/Credential values, hashes, and passwords are redacted/);
    expect(screen.getByLabelText('Session expiry seconds')).toHaveValue(3600);
    expect(screen.getByRole('combobox', { name: 'Permission for operator' })).toHaveValue('*');
    expect(screen.queryByDisplayValue(/\$2[aby]\$/)).not.toBeInTheDocument();
    expect(screen.getByText(/Legacy authentication detected/)).toBeVisible();

    await user.click(screen.getByRole('button', { name: 'Save Security Center' }));
    const dialog = await screen.findByRole('dialog');
    expect(posted).toHaveLength(0);
    expect(within(dialog).getByText(/Before: legacy aliases with redacted credentials/)).toBeVisible();
    const acknowledgement = within(dialog).getByTestId('confirmation-dialog-ack');
    expect(acknowledgement).not.toBeChecked();
    await user.click(acknowledgement);
    await user.click(within(dialog).getByRole('button', { name: /confirm/i }));

    await waitFor(() => expect(posted).toHaveLength(1));
    expect(posted[0]).toMatchObject({
      adminAuth: { type: 'credentials', users: [{ username: 'operator', permissions: '*', password: '' }], sessionExpiryTime: 3600 },
      httpNodeAuth: { user: 'nodes', pass: '' },
      httpStaticAuth: { user: 'static', pass: '' },
      expectedRevision: 'current-revision',
    });
  });

  it('announces read-only mode and hides editable controls', async () => {
    server.use(http.get('/api/bootstrap/status', () => ok({ ...editableHostStatus, configuration: { ...editableHostStatus.configuration, editable: false } })));
    const user = userEvent.setup();
    renderView();
    await user.click(await screen.findByRole('button', { name: 'Authentication' }));
    expect(await screen.findByRole('status')).toHaveTextContent(/read-only/i);
    expect(screen.queryByRole('button', { name: 'Save Security Center' })).not.toBeInTheDocument();
  });

  it('does not resubmit a successfully applied surface with the next surface', async () => {
    server.use(http.get('/api/settings/raw', () => ok({ content: 'module.exports = {};', writable: true, revision: { fingerprint: 'current-revision', algorithm: 'sha256' } })));
    const user = userEvent.setup();
    renderView();
    await user.click(await screen.findByRole('button', { name: 'Authentication' }));

    await user.clear(screen.getAllByLabelText('Username')[0]);
    await user.type(screen.getAllByLabelText('Username')[0], 'operator-updated');
    await user.click(screen.getByRole('button', { name: 'Save Security Center' }));
    await waitFor(() => expect(posted).toHaveLength(1));
    expect(posted[0]).toMatchObject({ adminAuth: { users: [{ username: 'operator-updated' }] } });
    expect(posted[0]).not.toHaveProperty('httpNodeAuth');
    expect(posted[0]).not.toHaveProperty('httpStaticAuth');

    await user.clear(screen.getAllByLabelText('Username')[1]);
    await user.type(screen.getAllByLabelText('Username')[1], 'nodes-updated');
    await user.click(screen.getByRole('button', { name: 'Save Security Center' }));
    await waitFor(() => expect(posted).toHaveLength(2));
    expect(posted[1]).toMatchObject({ httpNodeAuth: { user: 'nodes-updated' } });
    expect(posted[1]).not.toHaveProperty('adminAuth');
    expect(posted[1]).not.toHaveProperty('httpStaticAuth');
  });
});
