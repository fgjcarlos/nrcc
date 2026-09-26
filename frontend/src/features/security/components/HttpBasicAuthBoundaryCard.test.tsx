import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { I18nProvider } from '@/i18n';
import { server } from '@/test/msw/server';
import { HttpBasicAuthBoundaryCard } from './HttpBasicAuthBoundaryCard';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

function renderBoundary(
  props: Omit<Parameters<typeof HttpBasicAuthBoundaryCard>[0], 'onApplied'> & {
    onApplied?: () => void;
  },
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <HttpBasicAuthBoundaryCard {...props} onApplied={props.onApplied ?? vi.fn()} />
      </I18nProvider>
    </QueryClientProvider>,
  );
}

describe('HttpBasicAuthBoundaryCard — httpNodeAuth', () => {
  it('renders the success palette when the surface is configured', () => {
    renderBoundary({
      surface: 'httpNodeAuth',
      config: { user: 'nodes', pass: '$2a$hash' },
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
    });
    const card = screen.getByTestId('boundary-httpNodeAuth');
    expect(card).toBeInTheDocument();
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-success/);
  });

  it('renders the danger palette when not editable', () => {
    renderBoundary({
      surface: 'httpNodeAuth',
      config: { user: 'nodes' },
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: false,
    });
    const card = screen.getByTestId('boundary-httpNodeAuth');
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-base-300/);
    expect(screen.getByTestId('boundary-httpNodeAuth-readonly')).toBeInTheDocument();
  });

  it('surfaces the bcrypt warning when the new password is not a hash', () => {
    renderBoundary({
      surface: 'httpNodeAuth',
      config: { user: 'nodes', pass: '$2a$hash' },
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
    });
    const passInput = screen.getByTestId('boundary-httpNodeAuth-pass');
    fireEvent.change(passInput, { target: { value: 'plain-text' } });
    // The bcrypt warning shows when the typed value is non-empty + non-hash.
    // After the change event the local state holds 'plain-text' which is not
    // a bcrypt hash, so the warning surfaces.
    expect(screen.getByTestId('boundary-httpNodeAuth-bcrypt-warning')).toBeInTheDocument();
  });

  it('submits the httpNodeAuth payload on save', async () => {
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
      surface: 'httpNodeAuth',
      config: null,
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
      onApplied,
    });
    const userInput = screen.getByTestId('boundary-httpNodeAuth-user');
    await user.type(userInput, 'nodes');
    const passInput = screen.getByTestId('boundary-httpNodeAuth-pass');
    await user.type(passInput, '$2a$hash');
    await user.click(screen.getByTestId('boundary-httpNodeAuth-save'));
    await waitFor(() => expect(onApplied).toHaveBeenCalled());
    expect((posted as { httpNodeAuth?: { user: string } }).httpNodeAuth?.user).toBe(
      'nodes',
    );
  });
});

describe('HttpBasicAuthBoundaryCard — httpStaticAuth', () => {
  it('renders with a different test id namespace', () => {
    renderBoundary({
      surface: 'httpStaticAuth',
      config: { user: 'static' },
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
    });
    expect(screen.getByTestId('boundary-httpStaticAuth')).toBeInTheDocument();
    expect(screen.getByTestId('boundary-httpStaticAuth-user')).toBeInTheDocument();
    expect(screen.getByTestId('boundary-httpStaticAuth-pass')).toBeInTheDocument();
  });

  it('shows the migration banner when rawSettingsContent carries the legacy alias', () => {
    renderBoundary({
      surface: 'httpStaticAuth',
      config: null,
      rawSettingsContent: 'module.exports = { staticAuth: { user: "x", pass: "[redacted]" } };',
      expectedRevision: 'rev',
      editable: true,
    });
    expect(screen.getByTestId('boundary-httpStaticAuth-legacy')).toBeInTheDocument();
  });

  it('renders the success palette and applies when the surface is configured', async () => {
    const user = userEvent.setup();
    const onApplied = vi.fn();
    server.use(
      http.post('/api/config/apply', async ({ request }) => {
        await request.json();
        return ok({ configuration: {}, document: {} });
      }),
    );
    renderBoundary({
      surface: 'httpStaticAuth',
      config: { user: 'static', pass: '$2b$hash' },
      rawSettingsContent: '',
      expectedRevision: 'rev',
      editable: true,
      onApplied,
    });
    const card = screen.getByTestId('boundary-httpStaticAuth');
    const chip = card.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-success/);
    await user.click(screen.getByTestId('boundary-httpStaticAuth-save'));
    await waitFor(() => expect(onApplied).toHaveBeenCalled());
  });
});
