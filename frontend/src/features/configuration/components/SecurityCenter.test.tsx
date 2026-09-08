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
      http.get('/api/settings/raw', () => ok({ content: 'module.exports = { nodeHttpAuth: { user: "nodes", pass: "[redacted]" } };', writable: true })),
      http.post('/api/config', async ({ request }) => {
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
});
