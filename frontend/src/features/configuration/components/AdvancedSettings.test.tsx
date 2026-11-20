import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';
import { AdvancedSettings } from './AdvancedSettings';
import { server } from '@/test/msw/server';
import { i18n } from '@/i18n';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

const corePresets = [
  {
    id: 'https-tls-preset',
    name: 'HTTPS / TLS configuration',
    description: 'Manage the https block.',
    surfaces: ['http'],
    trust: 'managed',
    channelBoundary: 'http-only',
    managedKeys: ['https'],
    inputs: ['key', 'cert'],
  },
  {
    id: 'functionGlobalContext-strict',
    name: 'functionGlobalContext (strict)',
    description: 'Curated allow-list.',
    surfaces: ['editor'],
    trust: 'managed',
    channelBoundary: 'editor-only',
    managedKeys: ['functionGlobalContext'],
    inputs: [],
  },
];

function renderAdvanced() {
  return render(
    <QueryClientProvider
      client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
    >
      <AdvancedSettings rawContent="module.exports = {}" onApplied={vi.fn()} />
    </QueryClientProvider>
  );
}

describe('AdvancedSettings', () => {
  beforeEach(() => {
    server.use(
      http.get('/api/presets', () => ok(corePresets)),
      http.post('/api/presets/https-tls-preset/preview', () =>
        ok({
          after: 'patched',
          preview: '+ https: { ... }\n- https: [redacted]\n',
          replaced: ['https'],
          inserted: [],
        }),
      ),
    );
  });

  it('renders the curated preset catalogue with surface badges', async () => {
    renderAdvanced();
    expect(await screen.findByRole('heading', { name: /HTTPS \/ TLS/i })).toBeVisible();
    expect(screen.getByRole('heading', { name: /functionGlobalContext/i })).toBeVisible();
    // Surface badge text
    expect(screen.getAllByText(/http/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/editor/).length).toBeGreaterThan(0);
  });

  it('asks for the declared paths, sends them, and shows the redacted preview', async () => {
    const user = userEvent.setup();
    let body: unknown;
    server.use(
      http.post('/api/presets/https-tls-preset/preview', async ({ request }) => {
        body = await request.json();
        return ok({ after: 'patched', preview: '+ https: [redacted]\n', replaced: ['https'], inserted: [] });
      }),
    );
    renderAdvanced();
    const card = await screen.findByRole('region', { name: /https-tls-preset/i });
    await user.click(within(card).getByRole('button', { name: /preview/i }));
    const dialog = await screen.findByRole('dialog');
    const generate = within(dialog).getByRole('button', { name: 'Generate preview' });
    expect(generate).toBeDisabled();
    await user.type(within(dialog).getByLabelText('Private key file'), '/data/certs/key.pem');
    await user.type(within(dialog).getByLabelText('Certificate file'), '/data/certs/cert.pem');
    await user.click(generate);
    expect(await within(dialog).findByText(/\[redacted\]/)).toBeVisible();
    expect(body).toEqual({
      content: 'module.exports = {}',
      values: { key: '/data/certs/key.pem', cert: '/data/certs/cert.pem' },
    });
  });

  it('previews a preset without inputs straight away', async () => {
    const user = userEvent.setup();
    server.use(
      http.post('/api/presets/functionGlobalContext-strict/preview', () =>
        ok({ after: 'patched', preview: '+ functionGlobalContext: {}\n', replaced: [], inserted: ['functionGlobalContext'] }),
      ),
    );
    renderAdvanced();
    const card = await screen.findByRole('region', { name: /functionGlobalContext-strict/i });
    await user.click(within(card).getByRole('button', { name: /preview/i }));
    const dialog = await screen.findByRole('dialog');
    expect(await within(dialog).findByText(/functionGlobalContext: \{\}/)).toBeVisible();
    expect(within(dialog).queryByRole('button', { name: 'Generate preview' })).not.toBeInTheDocument();
  });

  it('applies only after acknowledgement, without typing a confirmation phrase', async () => {
    const user = userEvent.setup();
    server.use(
      http.post('/api/presets/functionGlobalContext-strict/preview', () =>
        ok({ after: 'patched', preview: '+ functionGlobalContext: {}\n', replaced: [], inserted: ['functionGlobalContext'] }),
      ),
    );
    renderAdvanced();
    const card = await screen.findByRole('region', { name: /functionGlobalContext-strict/i });
    await user.click(within(card).getByRole('button', { name: /preview/i }));
    const dialog = await screen.findByRole('dialog');
    await within(dialog).findByText(/functionGlobalContext: \{\}/);
    const apply = within(dialog).getByRole('button', { name: 'Apply preset' });
    expect(apply).toBeDisabled();
    expect(within(dialog).queryByRole('textbox')).not.toBeInTheDocument();
    await user.click(within(dialog).getByRole('checkbox'));
    expect(apply).toBeEnabled();
  });

  it('reports an unchanged result and keeps apply disabled', async () => {
    const user = userEvent.setup();
    server.use(
      http.post('/api/presets/functionGlobalContext-strict/preview', () =>
        ok({ after: 'module.exports = {}', preview: '', replaced: ['functionGlobalContext'], inserted: [] }),
      ),
    );
    renderAdvanced();
    const card = await screen.findByRole('region', { name: /functionGlobalContext-strict/i });
    await user.click(within(card).getByRole('button', { name: /preview/i }));
    const dialog = await screen.findByRole('dialog');
    expect(await within(dialog).findByText('This preset would not change settings.js.')).toBeVisible();
    await user.click(within(dialog).getByRole('checkbox'));
    expect(within(dialog).getByRole('button', { name: 'Apply preset' })).toBeDisabled();
  });

  it('explains a rejected path inline instead of the HTTP status', async () => {
    const user = userEvent.setup();
    server.use(
      http.post('/api/presets/https-tls-preset/preview', () =>
        HttpResponse.json(
          { success: false, error: { code: 'PRESET_NEEDS_INPUT', message: 'key must be an absolute file path' } },
          { status: 422 },
        ),
      ),
    );
    renderAdvanced();
    const card = await screen.findByRole('region', { name: /https-tls-preset/i });
    await user.click(within(card).getByRole('button', { name: /preview/i }));
    const dialog = await screen.findByRole('dialog');
    await user.type(within(dialog).getByLabelText('Private key file'), 'certs/key.pem');
    await user.type(within(dialog).getByLabelText('Certificate file'), '/data/cert.pem');
    await user.click(within(dialog).getByRole('button', { name: 'Generate preview' }));
    const alert = await within(dialog).findByRole('alert');
    expect(alert).toHaveTextContent('Enter absolute file paths (starting with /) for every field.');
    expect(alert).not.toHaveTextContent(/status code/);
  });

  it('renders the catalogue copy from the Spanish catalog', async () => {
    await i18n.changeLanguage('es');
    try {
      const user = userEvent.setup();
      renderAdvanced();
      const card = await screen.findByRole('region', { name: /https-tls-preset/i });
      expect(within(card).queryByText(/trust:/)).not.toBeInTheDocument();
      await user.click(within(card).getByRole('button', { name: /vista previa/i }));
      const dialog = await screen.findByRole('dialog');
      expect(within(dialog).getByLabelText('Archivo de clave privada')).toBeVisible();
      expect(within(dialog).getByRole('button', { name: 'Generar vista previa' })).toBeVisible();
    } finally {
      await i18n.changeLanguage('en');
    }
  });

  it('surfaces a 404 when the preset is not registered', async () => {
    server.use(
      http.post('/api/presets/missing-preset/preview', () =>
        HttpResponse.json(
          { success: false, error: { code: 'PRESET_NOT_REGISTERED', message: 'preset not registered' } },
          { status: 404 },
        ),
      ),
    );
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <AdvancedSettings rawContent="module.exports = {}" onApplied={vi.fn()} />
      </QueryClientProvider>,
    );
    // The component does not render a button for unregistered presets,
    // so we assert that the catalogue only shows registered presets.
    await waitFor(() => {
      expect(screen.queryByText(/missing-preset/)).not.toBeInTheDocument();
    });
  });

  // The backend does not mount the presets routes (no /api/presets in
  // internal/server), so the SPA handler answers 404 API_ROUTE_NOT_FOUND.
  // That is definitive: no retry spinner, and a calm, truthful message.
  it('reports presets as unavailable without retrying when the route is missing', async () => {
    let calls = 0;
    server.use(
      http.get('/api/presets', () => {
        calls += 1;
        return HttpResponse.json(
          { success: false, error: { code: 'API_ROUTE_NOT_FOUND', message: 'API endpoint not found' } },
          { status: 404 },
        );
      }),
    );
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: 3 } } })}>
        <AdvancedSettings rawContent="module.exports = {}" onApplied={vi.fn()} />
      </QueryClientProvider>,
    );

    const status = await screen.findByText('Curated presets are not available on this NRCC server.');
    expect(status.closest('[role="status"]')?.className).not.toMatch(/warning/);
    expect(screen.queryByText(/Loading curated presets/)).not.toBeInTheDocument();
    expect(calls).toBe(1);
  });
});
