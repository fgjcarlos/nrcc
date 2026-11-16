import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { http, HttpResponse } from 'msw';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { toast } from 'sonner';
import { I18nProvider, i18n } from '@/i18n';
import { server } from '@/test/msw/server';
import { authUsersResponse, editableHostStatus, securityCenterConfig } from '@/test/msw/fixtures';
import enSecurity from '@/locales/en/security.json';
import esSecurity from '@/locales/es/security.json';
import { SecurityView } from './SecurityView';

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const ok = (data: unknown) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() });

const FORM_SURFACES = [
  { anchor: 'admin-auth', testId: 'boundary-admin-auth' },
  { anchor: 'http-node-auth', testId: 'boundary-httpNodeAuth' },
  { anchor: 'http-static-auth', testId: 'boundary-httpStaticAuth' },
  { anchor: 'dashboard-access', testId: 'boundary-dashboard-access' },
];

function setupMocks(config: Record<string, unknown> = securityCenterConfig) {
  server.use(
    http.get('/api/bootstrap/status', () => ok(editableHostStatus)),
    http.get('/api/config', () => ok(config)),
    http.get('/api/settings/raw', () =>
      ok({
        content: 'module.exports = {};',
        writable: true,
        revision: { fingerprint: 'current-revision', algorithm: 'sha256' },
      }),
    ),
    http.get('/api/auth/users', () => ok(authUsersResponse)),
    http.get('/api/dashboards/discovery', () =>
      ok({ packages: { state: 'available' }, flows: { state: 'available' }, flowFuse: [], uiBases: [] }),
    ),
  );
}

function renderView() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
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

const surface = (testId: string) => screen.getByTestId(testId);
const details = (testId: string) => surface(testId).querySelector('details') as HTMLDetailsElement;
const chip = (testId: string) => surface(testId).querySelector('[role="status"]') as HTMLElement;
const editToggle = (testId: string) => within(surface(testId)).getByText(/^(Edit|Editar)$/, { selector: 'summary' });

async function waitForLoadedForm() {
  await waitFor(() => expect(screen.getAllByTestId('boundary-admin-auth-username').length).toBeGreaterThan(0));
}

describe('SecurityView surfaces (T10)', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    window.location.hash = '';
    await i18n.changeLanguage('en');
  });
  afterEach(async () => {
    window.location.hash = '';
    await i18n.changeLanguage('en');
  });

  it('renders every surface collapsed under a stable anchor with its state chip', async () => {
    setupMocks({ uiPort: 1880, httpNodeAuth: { user: 'nodes' } });
    renderView();
    await waitFor(() => expect(surface('boundary-admin-auth')).toBeInTheDocument());

    for (const { anchor, testId } of FORM_SURFACES) {
      expect(surface(testId)).toHaveAttribute('id', anchor);
      expect(details(testId).open).toBe(false);
    }
    expect(surface('boundary-nrcc-access')).toHaveAttribute('id', 'nrcc-access');
    expect(screen.getByTestId('boundary-admin-auth-fieldset')).not.toBeVisible();
    expect(screen.getByTestId('boundary-httpNodeAuth-fieldset')).not.toBeVisible();

    // adminAuth off exposes the editor: amber. Other surfaces off: neutral.
    expect(chip('boundary-admin-auth')).toHaveTextContent('Off');
    expect(chip('boundary-admin-auth').className).toMatch(/bg-ds-warning/);
    expect(chip('boundary-httpNodeAuth')).toHaveTextContent('On');
    expect(chip('boundary-httpNodeAuth').className).toMatch(/bg-ds-success/);
    expect(chip('boundary-httpStaticAuth')).toHaveTextContent('Off');
    expect(chip('boundary-httpStaticAuth').className).toMatch(/bg-base-300/);
    expect(chip('boundary-dashboard-access')).toHaveTextContent('Not reported');
    expect(screen.queryByText(/configuration topology/i)).not.toBeInTheDocument();
  });

  it('opens and closes a surface form from its Edit disclosure', async () => {
    const user = userEvent.setup();
    setupMocks();
    renderView();
    await waitForLoadedForm();

    await user.click(editToggle('boundary-admin-auth'));
    expect(details('boundary-admin-auth').open).toBe(true);
    expect(screen.getByTestId('boundary-admin-auth-fieldset')).toBeVisible();
    expect(details('boundary-httpNodeAuth').open).toBe(false);

    await user.click(editToggle('boundary-admin-auth'));
    expect(details('boundary-admin-auth').open).toBe(false);
    expect(screen.getByTestId('boundary-admin-auth-fieldset')).not.toBeVisible();
  });

  it('expands and scrolls to the surface named by the URL hash', async () => {
    const scrollIntoView = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoView;
    window.location.hash = '#http-static-auth';
    setupMocks();
    renderView();
    await waitFor(() => expect(details('boundary-httpStaticAuth').open).toBe(true));
    expect(screen.getByTestId('boundary-httpStaticAuth-fieldset')).toBeVisible();
    expect(details('boundary-admin-auth').open).toBe(false);
    expect(scrollIntoView).toHaveBeenCalled();
    expect(scrollIntoView.mock.contexts).toContain(surface('boundary-httpStaticAuth'));
  });

  it('keeps typed values when a surface is collapsed and expanded again', async () => {
    const user = userEvent.setup();
    setupMocks();
    renderView();
    await waitForLoadedForm();

    await user.click(editToggle('boundary-httpNodeAuth'));
    const nodeUser = screen.getByTestId('boundary-httpNodeAuth-user');
    await user.clear(nodeUser);
    await user.type(nodeUser, 'kept-user');
    await user.type(screen.getByTestId('boundary-httpNodeAuth-pass'), 'kept-pass');
    await user.click(editToggle('boundary-httpNodeAuth'));
    await user.click(editToggle('boundary-httpNodeAuth'));

    expect(screen.getByTestId('boundary-httpNodeAuth-user')).toHaveValue('kept-user');
    expect(screen.getByTestId('boundary-httpNodeAuth-pass')).toHaveValue('kept-pass');
  });

  it('states the bcrypt storage rule once on the page', async () => {
    setupMocks();
    renderView();
    await waitForLoadedForm();
    const text = document.body.textContent ?? '';
    expect(text.match(/bcrypt/gi)).toHaveLength(1);
  });

  it('shows no English security copy in an es render, including apply errors', async () => {
    await i18n.changeLanguage('es');
    const user = userEvent.setup();
    setupMocks();
    server.use(
      http.post('/api/config/apply', () =>
        HttpResponse.json(
          { success: false, error: { code: 'SETTINGS_REVISION_CONFLICT', message: 'conflict' }, timestamp: '' },
          { status: 409 },
        ),
      ),
    );
    renderView();
    await waitForLoadedForm();

    await user.click(editToggle('boundary-httpNodeAuth'));
    await user.click(screen.getByRole('button', { name: esSecurity.httpNodeAuth.save }));
    await waitFor(() =>
      expect(screen.getByTestId('boundary-httpNodeAuth-error')).toHaveTextContent(
        esSecurity.applyErrors.revisionConflict,
      ),
    );
    expect(toast.error).toHaveBeenCalledWith(esSecurity.applyErrors.revisionConflict);

    // DashboardAccess lives in features/configuration and is checked there.
    const owned = [...document.querySelectorAll('[data-testid="security-view"] > *')].filter(
      (node) => node.id !== 'dashboard-access',
    );
    const ariaLabels = owned.flatMap((node) =>
      [...node.querySelectorAll('[aria-label]')].map((el) => el.getAttribute('aria-label') ?? ''),
    );
    const rendered = [...owned.map((node) => node.textContent ?? ''), ...ariaLabels].join('\n');

    const flatten = (value: unknown): string[] =>
      typeof value === 'string' ? [value] : Object.values(value as object).flatMap(flatten);
    const es = new Set(flatten(esSecurity));
    const englishOnly = flatten(enSecurity)
      .flatMap((value) => value.split(/\{\{[^}]+\}\}/))
      .map((part) => part.trim())
      .filter((part) => part.length > 3 && !es.has(part) && ![...es].some((value) => value.includes(part)));
    expect(englishOnly.filter((part) => rendered.includes(part))).toEqual([]);
    expect(rendered).not.toMatch(/\buser \d|\band\b|Not reported|Edit\b/);
  });
});
