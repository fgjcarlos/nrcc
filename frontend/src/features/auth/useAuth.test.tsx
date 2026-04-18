import { useLocation } from 'react-router-dom'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { APIRequestError, api } from '../../api'
import { createTestWrapper } from '../../test/utils'
import { useAuth } from './useAuth'

vi.mock('../../api', async () => {
  const actual = await vi.importActual<typeof import('../../api')>('../../api')

  return {
    ...actual,
    api: {
      ...actual.api,
      authStatus: vi.fn(),
      me: vi.fn(),
      login: vi.fn(),
      register: vi.fn(),
      logout: vi.fn(),
    },
  }
})

function UseAuthProbe() {
  const location = useLocation()
  const auth = useAuth()

  return (
    <>
      <div data-testid="pathname">{location.pathname}</div>
      <div data-testid="auth-mode">{auth.authMode}</div>
      <div data-testid="auth-message">{auth.authMessage}</div>
      <div data-testid="loading">{String(auth.isLoading)}</div>
      <button onClick={() => auth.loginMutation.mutate({ username: 'admin', password: 'secret' })} type="button">
        Trigger login
      </button>
    </>
  )
}

describe('useAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('switches to register mode when the backend reports no users', async () => {
    vi.mocked(api.authStatus).mockResolvedValue({ hasUsers: false })
    vi.mocked(api.me).mockRejectedValue(new APIRequestError('Unauthorized', 401))

    render(<UseAuthProbe />, { wrapper: createTestWrapper('/login') })

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false')
    })

    expect(screen.getByTestId('auth-mode')).toHaveTextContent('register')
    expect(screen.getByTestId('pathname')).toHaveTextContent('/login')
  })

  it('redirects authenticated users away from the login page', async () => {
    vi.mocked(api.authStatus).mockResolvedValue({ hasUsers: true })
    vi.mocked(api.me).mockResolvedValue({
      user: { id: '1', username: 'admin', role: 'admin', createdAt: '2026-01-01T00:00:00Z' },
      csrfToken: 'csrf-123',
    })

    render(<UseAuthProbe />, { wrapper: createTestWrapper('/login') })

    await waitFor(() => {
      expect(screen.getByTestId('pathname')).toHaveTextContent('/app/overview')
    })
  })

  it('redirects unauthenticated users away from protected routes', async () => {
    vi.mocked(api.authStatus).mockResolvedValue({ hasUsers: true })
    vi.mocked(api.me).mockRejectedValue(new APIRequestError('Unauthorized', 401))

    render(<UseAuthProbe />, { wrapper: createTestWrapper('/app/overview') })

    await waitFor(() => {
      expect(screen.getByTestId('pathname')).toHaveTextContent('/login')
    })
  })

  it('surfaces backend login errors in authMessage', async () => {
    const user = userEvent.setup()

    vi.mocked(api.authStatus).mockResolvedValue({ hasUsers: true })
    vi.mocked(api.me).mockRejectedValue(new APIRequestError('Unauthorized', 401))
    vi.mocked(api.login).mockRejectedValue(new APIRequestError('Invalid credentials', 401, 'INVALID_CREDENTIALS'))

    render(<UseAuthProbe />, { wrapper: createTestWrapper('/login') })

    await waitFor(() => {
      expect(screen.getByTestId('loading')).toHaveTextContent('false')
    })

    await user.click(screen.getByRole('button', { name: 'Trigger login' }))

    await waitFor(() => {
      expect(screen.getByTestId('auth-message')).toHaveTextContent('Invalid credentials')
    })
  })
})
