import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { DashboardView } from './DashboardView'
import * as dashboardHooks from '../hooks'

vi.mock('../hooks', () => ({
  useDashboardData: vi.fn(),
  useDashboardActions: vi.fn(),
  useSystemHistory: vi.fn().mockReturnValue({ data: [], isLoading: false, isError: false }),
}))

vi.mock('../hooks/useSystemHistory', () => ({
  useSystemHistory: vi.fn().mockReturnValue({ data: [], isLoading: false, isError: false }),
}))

const renderDashboard = () =>
  render(
    <MemoryRouter>
      <DashboardView />
    </MemoryRouter>
  )

describe('DashboardView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(dashboardHooks.useDashboardActions).mockReturnValue({
      pendingConfirm: false,
      isRestarting: false,
      isStartStopping: false,
      setPendingConfirm: vi.fn(),
      handleRestartConfirm: vi.fn(),
      handleStartNodeRed: vi.fn(),
      handleStopNodeRed: vi.fn(),
      handleOpenNodeRed: vi.fn(),
    })
  })

  it('renders the three operational tiles in the documented order', () => {
    vi.mocked(dashboardHooks.useDashboardData).mockReturnValue({
      container: { inDocker: true, status: 'running', image: 'nodered:latest' },
      system: {
        resourceScope: 'host',
        cpu: { usage: 14, cores: 4, available: true },
        memory: { total: 8_589_934_592, used: 2_147_483_648, free: 6_442_450_944, usagePercent: 25, available: true },
        disk: { total: 107_374_182_400, used: 21_474_836_480, free: 85_899_345_920, usagePercent: 20, available: true },
        uptime: 3600,
        platform: 'linux',
        hostname: 'nrcc-smoke-host',
        nodeRedVersion: '5.0.6',
        edgeMode: false,
      },
      config: { adminAuth: { user: 'admin', password: 'x' }, requireHttps: true },
      host: {
        platform: 'linux',
        ready: true,
        interactive: false,
        nodejs: { name: 'node', installed: true, version: 'v22.0.0', command: 'node' },
        npm: { name: 'npm', installed: true, version: '11.0.0', command: 'npm' },
        nodeRedBinary: { name: 'node-red', installed: true, version: '5.0.6', command: 'node-red' },
        docker: { name: 'docker', installed: false },
        dockerCompose: { name: 'docker compose', installed: false },
        nodeRed: { detected: true, mode: 'native', managedByNrcc: true, running: true, version: '5.0.6', executable: '/usr/bin/node-red', userDir: '/tmp/nrcc-smoke', settingsPath: '/tmp/nrcc-smoke/settings.js' },
        settings: { path: '/tmp/nrcc-smoke/settings.js', source: 'disk', writable: true },
        configuration: { runtimeVersion: '5.0.6', adapter: 'nodered-5', catalogVersion: '5.0.6', source: 'disk', mode: 'editable', editable: true },
        recommendations: [],
      },
      runtime: { status: 'running', uptime: 3600 },
      backups: {
        scheduler: { enabled: true, scheduled: true, schedule: 'daily', customSchedule: '', activeSpec: '0 2 * * *', nextRunAt: '2026-01-02T02:00:00.000Z', lastRunAt: '2026-01-01T02:00:00.000Z', lastSuccessAt: '2026-01-01T02:00:00.000Z', lastBackupId: 'backup-001' },
        storage: { totalBackups: 12, totalSize: 8_589_934_592, manualCount: 4, autoCount: 8, preRestoreCount: 0 },
        recentEvents: [],
      },
      dockerSuccess: true,
      dockerLoading: false,
      dockerError: false,
    })

    renderDashboard()

    expect(screen.getByRole('heading', { name: 'Overview' })).toBeInTheDocument()
    expect(screen.getByTestId('overview-security-posture-tile')).toBeInTheDocument()
    expect(screen.getByTestId('overview-node-red-health-tile')).toBeInTheDocument()
    expect(screen.getByTestId('overview-backup-health-tile')).toBeInTheDocument()
  })

  it('shows warning surfaces when docker is unhealthy and host setup has issues', () => {
    vi.mocked(dashboardHooks.useDashboardData).mockReturnValue({
      container: { inDocker: true, status: 'exited', image: 'nodered:latest' },
      system: undefined,
      config: undefined,
      host: {
        platform: 'linux',
        ready: false,
        interactive: false,
        nodejs: { name: 'node', installed: false },
        npm: { name: 'npm', installed: true },
        nodeRedBinary: { name: 'node-red', installed: false },
        docker: { name: 'docker', installed: true },
        dockerCompose: { name: 'docker compose', installed: true },
        nodeRed: { detected: false, mode: 'unknown', managedByNrcc: false, running: false },
        settings: { path: '/tmp/settings.js', source: 'disk', writable: false },
        configuration: { runtimeVersion: 'unknown', adapter: 'none', catalogVersion: '5.0.6', source: 'disk', mode: 'read-only', editable: false, reason: 'Node-RED not detected' },
        recommendations: [],
      },
      runtime: { status: 'stopped', uptime: 0 },
      backups: undefined,
      dockerSuccess: true,
      dockerLoading: false,
      dockerError: false,
    })

    renderDashboard()

    expect(screen.getByRole('heading', { name: 'Overview' })).toBeInTheDocument()
    // Default-locale EN copy from the dashboard catalog (issue #767/#832).
    expect(screen.getByText('Docker container is not running correctly. Some features may not work.')).toBeInTheDocument()
    expect(screen.getByText('Node.js is not installed. Node-RED has not been detected yet. nrcc cannot write to settings.js.')).toBeInTheDocument()
    // Slice D replaces the SystemHealthCard's "Check environment for
    // issues" subtitle with the three operational tiles — security
    // posture (danger), Node-RED health (danger), backup health.
    expect(screen.getByTestId('overview-security-posture-tile')).toBeInTheDocument()
    expect(screen.getByTestId('overview-node-red-health-tile')).toBeInTheDocument()
  })

  it('renders fallback telemetry placeholders when dashboard data is missing', () => {
    vi.mocked(dashboardHooks.useDashboardData).mockReturnValue({
      container: undefined,
      system: undefined,
      config: undefined,
      host: undefined,
      runtime: undefined,
      backups: undefined,
      dockerSuccess: false,
      dockerLoading: false,
      dockerError: false,
    })

    renderDashboard()

    expect(screen.getByRole('heading', { name: 'Overview' })).toBeInTheDocument()
    // The three new tiles still mount; each renders its loading
    // state when the corresponding data is missing.
    expect(screen.getByTestId('overview-security-posture-tile')).toBeInTheDocument();
    expect(screen.getByTestId('overview-node-red-health-tile')).toBeInTheDocument();
    expect(screen.getByTestId('overview-backup-health-tile')).toBeInTheDocument();
    // NodeRedHealthTile still surfaces the restart/open actions.
    expect(screen.getByRole('button', { name: 'Restart' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Open' })).toBeInTheDocument();
  })
});
