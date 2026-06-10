import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { DockerView } from './DockerView'
import * as dockerHooks from '@/features/docker/hooks'

vi.mock('@/features/docker/hooks', () => ({
  useDockerData: vi.fn(),
  useDockerActions: vi.fn(),
}))

const mockRestartMutate = vi.fn()
const mockStopMutate = vi.fn()

const defaultActions = {
  restartMutation: { isPending: false, mutate: mockRestartMutate },
  stopMutation: { isPending: false, mutate: mockStopMutate },
}

describe('DockerView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(dockerHooks.useDockerActions).mockReturnValue(defaultActions as unknown as ReturnType<typeof dockerHooks.useDockerActions>)
  })

  it('shows a loading state while container status is loading', () => {
    vi.mocked(dockerHooks.useDockerData).mockReturnValue({
      container: undefined,
      isLoading: true,
      isError: false,
      error: null,
    })

    render(<DockerView />)

    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('shows an error state when the docker status query fails', () => {
    vi.mocked(dockerHooks.useDockerData).mockReturnValue({
      container: undefined,
      isLoading: false,
      isError: true,
      error: new Error('Docker unavailable'),
    })

    render(<DockerView />)

    expect(screen.getByText('An error occurred')).toBeInTheDocument()
  })

  it('shows an empty state when no container data is available', () => {
    vi.mocked(dockerHooks.useDockerData).mockReturnValue({
      container: undefined,
      isLoading: false,
      isError: false,
      error: null,
    })

    render(<DockerView />)

    expect(screen.getByText('No container data available')).toBeInTheDocument()
  })

  it('renders the container details and restart flow', async () => {
    const user = userEvent.setup()

    vi.mocked(dockerHooks.useDockerData).mockReturnValue({
      container: {
        id: 'abc123',
        name: 'nodered',
        image: 'nodered:latest',
        status: 'running',
        created: '2026-05-22T00:00:00Z',
        ports: [{ publicPort: 1880, privatePort: 1880, type: 'tcp' }],
        state: {
          running: true,
          paused: false,
          restartCount: 2,
          memory: 1048576,
          cpu: 0.2,
        },
      },
      isLoading: false,
      isError: false,
      error: null,
    })

    render(<DockerView />)

    expect(screen.getByRole('heading', { name: 'Docker' })).toBeInTheDocument()
    expect(screen.getByText('nodered')).toBeInTheDocument()
    expect(screen.getAllByText('1880').length).toBeGreaterThan(0)

    await user.click(screen.getByRole('button', { name: /restart container/i }))
    expect(screen.getByRole('heading', { name: /restart container/i })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /confirm/i }))
    expect(mockRestartMutate).toHaveBeenCalled()
  })
})
