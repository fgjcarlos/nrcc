import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { App } from './App'
import { api } from './api'
import { useAuth } from './features/auth/useAuth'

vi.mock('./api', async () => {
  const actual = await vi.importActual<typeof import('./api')>('./api')

  return {
    ...actual,
    api: {
      ...actual.api,
      runtimeStatus: vi.fn(),
      runtimeLogs: vi.fn(),
      systemInfo: vi.fn(),
      environment: vi.fn(),
      backups: vi.fn(),
      libraries: vi.fn(),
      operationsStatus: vi.fn(),
      updateStatus: vi.fn(),
      diagnosticsReport: vi.fn(),
      diagnosticsLogs: vi.fn(),
      diagnosticsJobs: vi.fn(),
    },
  }
})

vi.mock('./features/auth/useAuth', () => ({
  useAuth: vi.fn(),
}))

const defaultUser = {
  id: '1',
  username: 'admin',
  role: 'admin',
  createdAt: '2026-01-01T00:00:00Z',
}

function createMockAuthState(overrides: Partial<ReturnType<typeof useAuth>> = {}) {
  return {
    user: defaultUser,
    isLoading: false,
    authMode: 'login',
    setAuthMode: vi.fn(),
    authMessage: '',
    loginMutation: { isPending: false, isSuccess: false, mutate: vi.fn() },
    registerMutation: { isPending: false, isSuccess: false, mutate: vi.fn() },
    logoutMutation: { isPending: false, isSuccess: false, mutate: vi.fn() },
    ...overrides,
  } as unknown as ReturnType<typeof useAuth>
}

function renderApp(route: string) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>
        <App />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('App routing', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    vi.mocked(useAuth).mockReturnValue(createMockAuthState())

    vi.mocked(api.runtimeStatus).mockResolvedValue({
      running: true,
      healthy: true,
      pid: 123,
      port: 1880,
      uptimeSec: 3600,
      version: '4.0.0',
      dataDir: '/var/lib/node-red',
      startedAt: '2026-01-01T00:00:00Z',
    })
    vi.mocked(api.runtimeLogs).mockResolvedValue({ lines: ['runtime ready'] })
    vi.mocked(api.systemInfo).mockResolvedValue({
      goos: 'linux',
      goarch: 'amd64',
      cpus: 4,
      hostname: 'nrcc-host',
      timestamp: '2026-01-01T00:00:00Z',
      localAccess: {
        mode: 'portless',
        hostname: 'nrcc.localhost',
        url: 'https://nrcc.localhost',
        fallbackUrl: 'http://127.0.0.1:3000',
        portlessAvailable: true,
        configured: true,
        operational: true,
        message: 'Stable local hostname configured at https://nrcc.localhost',
      },
    })
    vi.mocked(api.environment).mockResolvedValue({
      variables: [{ name: 'NODE_ENV', value: 'production' }],
      restartRequired: false,
    })
    vi.mocked(api.backups).mockResolvedValue({
      items: [
        {
          id: 'backup-1',
          reason: 'Nightly backup',
          createdAt: '2026-01-01T00:00:00Z',
          archiveName: 'backup-1.tar.gz',
          archiveBytes: 1024,
          archiveSha256: 'abc123',
        },
      ],
    })
    vi.mocked(api.libraries).mockResolvedValue({
      items: [{ name: '@node-red/dashboard', version: '1.0.0', direct: true }],
    })
    vi.mocked(api.operationsStatus).mockResolvedValue({ busy: false })
    vi.mocked(api.updateStatus).mockResolvedValue({
      installedVersion: '1.0.0',
      availableVersion: '1.0.1',
      updateAvailable: true,
    })
    vi.mocked(api.diagnosticsReport).mockResolvedValue({
      generatedAt: '2026-01-01T00:00:00Z',
      overallStatus: 'healthy',
      checks: [{ id: 'runtime', label: 'Runtime', severity: 'warning', status: 'pass', message: 'Runtime healthy' }],
    })
    vi.mocked(api.diagnosticsLogs).mockResolvedValue({
      logs: [{ timestamp: '2026-01-01T00:00:00Z', level: 'info', source: 'runtime', message: 'All good' }],
      total: 1,
    })
    vi.mocked(api.diagnosticsJobs).mockResolvedValue({
      jobs: [{ id: 'job-1', type: 'backup', status: 'completed', started_at: '2026-01-01T00:00:00Z' }],
      total: 1,
    })
  })

  it('renders the overview dashboard for authenticated users', async () => {
    renderApp('/app/overview')

    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Logs' })).toBeInTheDocument()
  })

  it('supports navigation to another primary page from the shell', async () => {
    const user = userEvent.setup()

    renderApp('/app/overview')

    await screen.findByRole('heading', { name: 'Dashboard' })
    const primaryNavigation = screen.getByRole('navigation', { name: 'Primary' })
    await user.click(within(primaryNavigation).getByRole('link', { name: 'Libraries' }))

    expect(await screen.findByRole('heading', { name: 'Libraries' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Install package' })).toBeInTheDocument()
  })

  it('redirects unknown protected routes back to the overview page', async () => {
    renderApp('/app/does-not-exist')

    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
  })

  it('renders another representative primary page directly by route', async () => {
    renderApp('/app/diagnostics')

    expect(await screen.findByRole('heading', { name: 'Diagnostics' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Doctor' })).toHaveAttribute('aria-selected', 'true')
  })

  it('redirects unauthenticated users to login from protected routes', async () => {
    vi.mocked(useAuth).mockReturnValue(createMockAuthState({ user: undefined }))

    renderApp('/app/overview')

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Sign in to the local control center' })).toBeInTheDocument()
    })
  })

  it('disables restart actions while another operation is in progress', async () => {
    vi.mocked(api.operationsStatus).mockResolvedValue({ busy: true, type: 'updating', detail: 'node-red' })

    renderApp('/app/overview')

    expect(await screen.findByRole('heading', { name: 'Dashboard' })).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getAllByRole('button', { name: /restart/i })[0]).toBeDisabled()
    })
    expect(screen.getAllByText(/updating in progress: node-red/i).length).toBeGreaterThan(0)
  })

  it('disables backup restore actions while another operation is in progress', async () => {
    vi.mocked(api.operationsStatus).mockResolvedValue({ busy: true, type: 'restarting', detail: 'node-red' })

    renderApp('/app/backups')

    expect(await screen.findByRole('heading', { name: 'Backups' })).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Restore' })).toBeDisabled()
    })
    expect(screen.getByText(/restarting in progress: node-red/i)).toBeInTheDocument()
  })
})
