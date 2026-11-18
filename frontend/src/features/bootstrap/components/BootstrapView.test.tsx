import { render, screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { i18n } from '@/i18n';
import { useEnvironment } from '@/features/bootstrap/hooks/useEnvironment';
import type { HostStatusWithCodes } from '@/features/bootstrap/types';
import { BootstrapView } from './BootstrapView';

vi.mock('@/features/bootstrap/hooks/useEnvironment');

// Legacy backend strings (Spanish, not localized) that codes must replace.
const PORTLESS_ES = 'Opcional: instala Portless para exponer nrcc o Node-RED con URLs HTTPS .localhost, LAN o Tailscale.';
const WRITABLE_ES = 'Otorga permisos de escritura sobre settings.js para que nrcc pueda guardar cambios.';

// Shape of the real NRCC container: the bundled binary resolves the native
// default under $HOME, while NODE_RED_SETTINGS points NRCC at /data.
const containerStatus: HostStatusWithCodes = {
  platform: 'linux',
  ready: true,
  interactive: false,
  nodejs: { name: 'node', installed: true, version: 'v22.0.0' },
  npm: { name: 'npm', installed: true, version: '11.0.0' },
  nodeRedBinary: { name: 'node-red', installed: true, version: '5.0.6' },
  docker: { name: 'docker', installed: false },
  dockerCompose: { name: 'docker compose', installed: false },
  nodeRed: {
    detected: true,
    mode: 'native',
    managedByNrcc: false,
    running: true,
    version: '5.0.6',
    userDir: '/usr/src/node-red/.node-red',
    settingsPath: '/usr/src/node-red/.node-red/settings.js',
  },
  settings: { path: '/data/settings.js', source: 'env', writable: true },
  configuration: {
    runtimeVersion: '5.0.6',
    adapter: 'nodered-5',
    catalogVersion: '5.0.6',
    source: 'env',
    mode: 'editable',
    editable: true,
  },
  recommendations: [PORTLESS_ES, WRITABLE_ES],
  recommendationCodes: ['portless_optional', 'settings_not_writable'],
};

function mockEnvironment(overrides: Partial<ReturnType<typeof useEnvironment>> = {}) {
  vi.mocked(useEnvironment).mockReturnValue({
    data: containerStatus,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
    ...overrides,
  } as unknown as ReturnType<typeof useEnvironment>);
}

/** The value a <dt> labels, read through the dt/dd pair the user sees. */
function valueOf(label: string) {
  const term = screen.getByText(label, { selector: 'dt' });
  return term.nextElementSibling as HTMLElement;
}

describe('BootstrapView', () => {
  beforeEach(() => mockEnvironment());
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it('titles the page without a kicker above the heading', () => {
    render(<BootstrapView />);
    expect(screen.getByRole('heading', { level: 1, name: 'Host & environment' })).toBeInTheDocument();
    expect(screen.queryByText('System')).not.toBeInTheDocument();
  });

  it('renders recommendations from their codes in English', () => {
    render(<BootstrapView />);
    expect(screen.getByText('Optional: install Portless to reach NRCC or Node-RED through HTTPS .localhost, LAN or Tailscale URLs.')).toBeInTheDocument();
    expect(screen.getByText('Grant write access to settings.js so NRCC can save changes.')).toBeInTheDocument();
    expect(screen.queryByText(PORTLESS_ES)).not.toBeInTheDocument();
    expect(screen.queryByText(WRITABLE_ES)).not.toBeInTheDocument();
  });

  it('renders recommendations from their codes in Spanish', async () => {
    await i18n.changeLanguage('es');
    render(<BootstrapView />);
    expect(screen.getByRole('heading', { level: 1, name: 'Host y entorno' })).toBeInTheDocument();
    expect(screen.getByText('Opcional: instala Portless para acceder a NRCC o a Node-RED con URLs HTTPS .localhost, de la LAN o de Tailscale.')).toBeInTheDocument();
    expect(screen.getByText('Concede permiso de escritura sobre settings.js para que NRCC pueda guardar cambios.')).toBeInTheDocument();
  });

  it('falls back to the backend string for an unknown code or a backend without codes', () => {
    mockEnvironment({
      data: { ...containerStatus, recommendations: ['Backend says hello'], recommendationCodes: ['future_code'] },
    } as Partial<ReturnType<typeof useEnvironment>>);
    const { unmount } = render(<BootstrapView />);
    expect(screen.getByText('Backend says hello')).toBeInTheDocument();
    unmount();

    mockEnvironment({
      data: { ...containerStatus, recommendations: ['Older backend text'], recommendationCodes: undefined },
    } as Partial<ReturnType<typeof useEnvironment>>);
    render(<BootstrapView />);
    expect(screen.getByText('Older backend text')).toBeInTheDocument();
  });

  it('shows catalog copy instead of the raw error message, with a retry', async () => {
    const refetch = vi.fn();
    mockEnvironment({ data: undefined, error: new Error('Request failed with status code 500'), refetch } as Partial<ReturnType<typeof useEnvironment>>);
    render(<BootstrapView />);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveTextContent('Could not load host details');
    expect(alert).toHaveTextContent('NRCC did not answer the host status request. Check that the NRCC service is running, then try again.');
    expect(screen.queryByText(/status code 500/)).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Try again' }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it('labels the install default and the edited settings.js for what each one is', () => {
    render(<BootstrapView />);
    expect(valueOf('Install default settings.js')).toHaveTextContent('/usr/src/node-red/.node-red/settings.js');
    expect(screen.getByText('Where the detected install keeps settings.js by default. NRCC edits the file below instead.')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: 'settings.js NRCC edits' })).toBeInTheDocument();
    expect(valueOf('Path')).toHaveTextContent('/data/settings.js');
    expect(valueOf('Chosen by')).toHaveTextContent('NODE_RED_SETTINGS environment variable');
  });

  it('drops the "edits instead" hint when both paths are the same file', () => {
    mockEnvironment({
      data: { ...containerStatus, settings: { ...containerStatus.settings, path: '/usr/src/node-red/.node-red/settings.js', source: 'detected' } },
    } as Partial<ReturnType<typeof useEnvironment>>);
    render(<BootstrapView />);
    expect(screen.queryByText(/NRCC edits the file below instead/)).not.toBeInTheDocument();
    expect(valueOf('Chosen by')).toHaveTextContent('Default of the detected install');
  });

  it('shows paths as mono designators that break only after a slash', () => {
    render(<BootstrapView />);
    const path = valueOf('Path').querySelector('code');
    expect(path).toHaveClass('font-mono');
    expect(path).not.toHaveClass('break-all');
    expect(path?.querySelectorAll('wbr')).toHaveLength(2);
  });

  it('states mode neutrally and running, write access and readiness as real states', () => {
    render(<BootstrapView />);
    expect(screen.getByRole('status', { name: 'Native install' })).toHaveClass('text-base-content/80');
    expect(screen.getByRole('status', { name: 'Running' })).toHaveClass('text-ds-on-tint-success');
    expect(screen.getByRole('status', { name: 'Writable' })).toHaveClass('text-ds-on-tint-success');
    expect(screen.getByRole('status', { name: 'Ready' })).toHaveClass('text-ds-on-tint-success');
  });

  it('marks a stopped runtime and a read-only settings.js as warnings, not errors', () => {
    mockEnvironment({
      data: {
        ...containerStatus,
        ready: false,
        nodeRed: { ...containerStatus.nodeRed, running: false },
        settings: { ...containerStatus.settings, writable: false },
      },
    } as Partial<ReturnType<typeof useEnvironment>>);
    render(<BootstrapView />);
    expect(screen.getByRole('status', { name: 'Not running' })).toHaveClass('text-ds-on-tint-warning');
    expect(screen.getByRole('status', { name: 'Not writable' })).toHaveClass('text-ds-on-tint-warning');
    expect(screen.getByRole('status', { name: 'Not ready' })).toHaveClass('text-ds-on-tint-warning');
  });

  it('uses the shared skeleton while loading', () => {
    mockEnvironment({ data: undefined, isLoading: true } as Partial<ReturnType<typeof useEnvironment>>);
    render(<BootstrapView />);
    expect(screen.getByRole('status', { name: 'Loading...' })).toHaveAttribute('aria-busy', 'true');
  });

  it('hides every icon from assistive technology', () => {
    const { container } = render(<BootstrapView />);
    for (const svg of container.querySelectorAll('svg')) {
      expect(svg.closest('[aria-hidden="true"]')).not.toBeNull();
    }
  });
});
