import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { I18nProvider } from '@/i18n';
import { NodeRedHealthTile } from './NodeRedHealthTile';
import type { HostStatus, RuntimeInfo, SystemInfo } from '@/shared/types';

function buildHost(overrides: Partial<HostStatus> = {}): HostStatus {
  return {
    platform: 'linux',
    ready: true,
    interactive: false,
    nodejs: { name: 'node', installed: true, version: 'v22.0.0', command: 'node' },
    npm: { name: 'npm', installed: true, version: '11.0.0', command: 'npm' },
    nodeRedBinary: { name: 'node-red', installed: true, version: '5.0.6', command: 'node-red' },
    docker: { name: 'docker', installed: false },
    dockerCompose: { name: 'docker compose', installed: false },
    nodeRed: {
      detected: true,
      mode: 'native',
      managedByNrcc: true,
      running: true,
      version: '5.0.6',
      executable: '/usr/bin/node-red',
      userDir: '/tmp/nrcc-smoke',
      settingsPath: '/tmp/nrcc-smoke/settings.js',
    },
    settings: {
      path: '/tmp/nrcc-smoke/settings.js',
      source: 'disk',
      writable: true,
    },
    configuration: {
      runtimeVersion: '5.0.6',
      adapter: 'nodered-5',
      catalogVersion: '5.0.6',
      source: 'disk',
      mode: 'editable',
      editable: true,
    },
    recommendations: [],
    ...overrides,
  };
}

function buildSystem(overrides: Partial<SystemInfo> = {}): SystemInfo {
  return {
    resourceScope: 'host',
    cpu: { usage: 14, cores: 4, available: true },
    memory: { total: 8_589_934_592, used: 2_147_483_648, free: 6_442_450_944, usagePercent: 25, available: true },
    disk: { total: 107_374_182_400, used: 21_474_836_480, free: 85_899_345_920, usagePercent: 20, available: true },
    uptime: 3600,
    platform: 'linux',
    hostname: 'nrcc-smoke-host',
    nodeRedVersion: '5.0.6',
    edgeMode: false,
    ...overrides,
  };
}

function buildRuntime(overrides: Partial<RuntimeInfo> = {}): RuntimeInfo {
  return {
    status: 'running',
    uptime: 3600,
    ...overrides,
  };
}

function renderTile(props: Parameters<typeof NodeRedHealthTile>[0]) {
  return render(
    <I18nProvider>
      <MemoryRouter>
        <NodeRedHealthTile {...props} />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe('NodeRedHealthTile', () => {
  it('renders the success palette when runtime is running and host is ready', () => {
    renderTile({
      host: buildHost(),
      runtime: buildRuntime(),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: vi.fn(),
      onOpenNodeRed: vi.fn(),
    });
    const tile = screen.getByTestId('overview-node-red-health-tile');
    const chip = tile.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-success/);
  });

  it('renders the warning palette when the runtime is stopped', () => {
    renderTile({
      host: buildHost(),
      runtime: buildRuntime({ status: 'stopped' }),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: vi.fn(),
      onOpenNodeRed: vi.fn(),
    });
    const tile = screen.getByTestId('overview-node-red-health-tile');
    const chip = tile.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-warning/);
  });

  it('renders the danger palette when the host is not ready', () => {
    renderTile({
      host: buildHost({ ready: false }),
      runtime: buildRuntime(),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: vi.fn(),
      onOpenNodeRed: vi.fn(),
    });
    const tile = screen.getByTestId('overview-node-red-health-tile');
    const chip = tile.querySelector('[role="status"]');
    expect(chip?.className).toMatch(/bg-ds-danger/);
  });

  it('disables the restart button when no Node-RED is detected', () => {
    renderTile({
      host: buildHost({ nodeRed: { ...buildHost().nodeRed, detected: false } }),
      runtime: buildRuntime(),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: vi.fn(),
      onOpenNodeRed: vi.fn(),
    });
    expect(screen.getByTestId('overview-node-red-restart')).toBeDisabled();
  });

  it('invokes onRequestRestart when the restart button is clicked', () => {
    const onRestart = vi.fn();
    renderTile({
      host: buildHost(),
      runtime: buildRuntime(),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: onRestart,
      onOpenNodeRed: vi.fn(),
    });
    screen.getByTestId('overview-node-red-restart').click();
    expect(onRestart).toHaveBeenCalledOnce();
  });

  it('invokes onOpenNodeRed when the open button is clicked', () => {
    const onOpen = vi.fn();
    renderTile({
      host: buildHost(),
      runtime: buildRuntime(),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: vi.fn(),
      onOpenNodeRed: onOpen,
    });
    screen.getByTestId('overview-node-red-open').click();
    expect(onOpen).toHaveBeenCalledOnce();
  });

  it('surfaces the deep-link to /configuration', () => {
    renderTile({
      host: buildHost(),
      runtime: buildRuntime(),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: vi.fn(),
      onOpenNodeRed: vi.fn(),
    });
    expect(screen.getByTestId('overview-node-red-link')).toHaveAttribute('href', '/configuration');
  });

  it('renders the three resource panels with the live percentages', () => {
    renderTile({
      host: buildHost(),
      runtime: buildRuntime(),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: vi.fn(),
      onOpenNodeRed: vi.fn(),
    });
    const panel = screen.getByTestId('overview-node-red-resources');
    expect(panel).toBeInTheDocument();
    expect(panel.textContent).toMatch(/25%/); // cpu 14% is in formatPercent — but assertion uses cpu.usage=14 → '14%'
  });

  it('shows the issues list when host is not ready', () => {
    renderTile({
      host: buildHost({
        ready: false,
        nodejs: { name: 'node', installed: false },
        nodeRedBinary: { name: 'node-red', installed: false },
      }),
      runtime: buildRuntime(),
      system: buildSystem(),
      inDocker: false,
      isRestarting: false,
      onRequestRestart: vi.fn(),
      onOpenNodeRed: vi.fn(),
    });
    expect(screen.getByTestId('overview-node-red-issues')).toBeInTheDocument();
  });
});
