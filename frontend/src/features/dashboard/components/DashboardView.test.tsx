import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { DashboardView } from './DashboardView'
import * as dashboardHooks from '../hooks'

vi.mock('../hooks', () => ({
  useDashboardData: vi.fn(),
  useDashboardActions: vi.fn(),
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
        recommendations: [],
      },
      backups: undefined,
      dockerSuccess: true,
      dockerLoading: false,
      dockerError: false,
    })

    renderDashboard()

    expect(screen.getByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
    expect(screen.getByText('Docker container is not running correctly. Some features may not work.')).toBeInTheDocument()
    expect(screen.getByText('Node.js no está instalado. Node-RED aún no fue detectado. nrcc no puede escribir sobre settings.js.')).toBeInTheDocument()
    expect(screen.getByText('Check environment for issues')).toBeInTheDocument()
  })

  it('renders fallback telemetry placeholders when dashboard data is missing', () => {
    vi.mocked(dashboardHooks.useDashboardData).mockReturnValue({
      container: undefined,
      system: undefined,
      config: undefined,
      host: undefined,
      backups: undefined,
      dockerSuccess: false,
      dockerLoading: false,
      dockerError: false,
    })

    renderDashboard()

    expect(screen.getByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
    expect(screen.getByText('Disk Usage')).toBeInTheDocument()
    expect(screen.getByText('Quick Actions')).toBeInTheDocument()
    expect(screen.getByText('Sin detectar')).toBeInTheDocument()
    expect(screen.getByText('Sin backups')).toBeInTheDocument()
    expect(screen.getByText('Cargando observabilidad')).toBeInTheDocument()
  })
})
