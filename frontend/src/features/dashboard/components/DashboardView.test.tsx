import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
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
    expect(screen.getByRole('button', { name: 'Open Node-RED editor' })).toBeInTheDocument();
  })
});

// T8 (ui-brand-overhaul): one honest summary sentence plus at most one
// next action, plain posture rows, no kicker, honest unavailable metrics.
describe('DashboardView — summary and next action', () => {
  const HOUR = 3_600_000
  const lockedConfig = {
    adminAuth: { type: 'credentials', users: [{ username: 'admin' }] },
    httpNodeAuth: { user: 'nodes', pass: 'x' },
    httpStaticAuth: { user: 'static', pass: 'x' },
    requireHttps: true,
  }
  const runningHost = {
    platform: 'linux',
    ready: true,
    interactive: false,
    nodejs: { name: 'node', installed: true },
    npm: { name: 'npm', installed: true },
    nodeRedBinary: { name: 'node-red', installed: true },
    docker: { name: 'docker', installed: false },
    dockerCompose: { name: 'docker compose', installed: false },
    nodeRed: { detected: true, mode: 'native', managedByNrcc: true, running: true, version: '5.0.7' },
    settings: { path: '/data/settings.js', source: 'disk', writable: true },
    configuration: { runtimeVersion: '5.0.7', adapter: 'nodered-5', catalogVersion: '5.0.7', source: 'disk', mode: 'editable', editable: true },
    recommendations: [],
  }
  const backupsWith = (latest: boolean) => ({
    scheduler: { enabled: true, scheduled: true, schedule: 'daily', customSchedule: '', activeSpec: '0 2 * * *', lastSuccessAt: '', lastBackupId: undefined },
    storage: { totalBackups: latest ? 1 : 0, totalSize: latest ? 1024 : 0, manualCount: latest ? 1 : 0, autoCount: 0, preRestoreCount: 0 },
    latestBackup: latest
      ? { id: 'backup-001', name: 'Manual backup', type: 'manual', createdAt: new Date(Date.now() - 3 * HOUR).toISOString(), totalSize: 1024 }
      : undefined,
    recentEvents: [],
  })
  const mockData = (overrides: Record<string, unknown>) =>
    vi.mocked(dashboardHooks.useDashboardData).mockReturnValue({
      container: undefined,
      system: undefined,
      config: lockedConfig,
      host: runningHost,
      runtime: { status: 'running', uptime: 60 },
      backups: backupsWith(true),
      dockerSuccess: false,
      dockerLoading: false,
      dockerError: false,
      ...overrides,
    } as unknown as ReturnType<typeof dashboardHooks.useDashboardData>)

  beforeEach(() => {
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

  it('exposed instance: says so and offers "Review security" linking to /security', () => {
    mockData({ config: { uiPort: 1880 } })
    renderDashboard()
    expect(screen.getByTestId('overview-summary')).toHaveTextContent(
      'Node-RED is running, the editor has no admin authentication and the last backup was 3 hours ago.',
    )
    const action = screen.getByRole('link', { name: 'Review security' })
    expect(action).toHaveAttribute('href', '/security')
    expect(screen.queryByRole('link', { name: 'Create a backup' })).not.toBeInTheDocument()
  })

  it('no backup yet: offers "Create a backup" linking to /backups', () => {
    mockData({ backups: backupsWith(false) })
    renderDashboard()
    expect(screen.getByTestId('overview-summary')).toHaveTextContent(
      'Node-RED is running, every authentication surface is on and there is no backup yet.',
    )
    expect(screen.getByRole('link', { name: 'Create a backup' })).toHaveAttribute('href', '/backups')
    expect(screen.queryByRole('link', { name: 'Review security' })).not.toBeInTheDocument()
  })

  it('healthy instance: states it and offers no summary action', () => {
    mockData({})
    renderDashboard()
    expect(screen.getByTestId('overview-summary')).toHaveTextContent(
      'Node-RED is running, every authentication surface is on and the last backup was 3 hours ago.',
    )
    expect(screen.queryByTestId('overview-summary-action')).not.toBeInTheDocument()
  })

  it('never invents a state when the data has not arrived', () => {
    mockData({ config: undefined, host: undefined, runtime: undefined, backups: undefined })
    renderDashboard()
    expect(screen.getByTestId('overview-summary')).toHaveTextContent(
      'Node-RED status is not reported, security is not reported and backups are not reported.',
    )
    expect(screen.queryByTestId('overview-summary-action')).not.toBeInTheDocument()
    expect(screen.queryByText('Exposed')).not.toBeInTheDocument()
  })

  it('has no kicker above the page title', () => {
    mockData({})
    renderDashboard()
    expect(screen.getByRole('heading', { level: 1, name: 'Overview' })).toBeInTheDocument()
    expect(screen.queryByText('System overview')).not.toBeInTheDocument()
    expect(screen.queryByText('Live telemetry')).not.toBeInTheDocument()
  })

  it('posture rows are plain, non-focusable rows read from the backend keys', () => {
    mockData({})
    renderDashboard()
    for (const key of ['adminAuth', 'httpNodeAuth', 'httpStaticAuth', 'requireHttps']) {
      const row = screen.getByTestId(`overview-security-${key}`)
      expect(row).toHaveTextContent('On')
      expect(row.closest('a, button, [tabindex]')).toBeNull()
    }
  })

  it('labels unavailable metrics as "Not reported" instead of a dash', () => {
    mockData({
      system: {
        resourceScope: 'container',
        cpu: { usage: 0, cores: 0, available: false },
        memory: { total: 0, used: 0, free: 0, usagePercent: 0, available: false },
        disk: { total: 0, used: 0, free: 0, usagePercent: 0, available: false },
        uptime: 0,
        platform: 'linux',
        hostname: 'h',
      },
    })
    renderDashboard()
    const metrics = screen.getByTestId('resource-metrics')
    expect(metrics).toHaveTextContent('Container scope')
    expect(within(metrics).getAllByText('Not reported')).toHaveLength(3)
    expect(metrics).not.toHaveTextContent('—')
  })

  it('names the editor action and keeps Restart behind its confirmation', () => {
    const setPendingConfirm = vi.fn()
    vi.mocked(dashboardHooks.useDashboardActions).mockReturnValue({
      pendingConfirm: false,
      isRestarting: false,
      isStartStopping: false,
      setPendingConfirm,
      handleRestartConfirm: vi.fn(),
      handleStartNodeRed: vi.fn(),
      handleStopNodeRed: vi.fn(),
      handleOpenNodeRed: vi.fn(),
    })
    mockData({})
    renderDashboard()
    expect(screen.getByRole('button', { name: 'Open Node-RED editor' })).toBeInTheDocument()
    screen.getByRole('button', { name: 'Restart' }).click()
    expect(setPendingConfirm).toHaveBeenCalledWith(true)
  })
})
