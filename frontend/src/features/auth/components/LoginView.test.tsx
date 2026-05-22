import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { LoginView } from './LoginView'
import { authService } from '../services/authService'
import * as useAuthModule from '../hooks/useAuth'
import { buildAuthMock } from '../__test-utils__/authMock'

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}))

vi.mock('../services/authService', () => ({
  authService: {
    getStatus: vi.fn(),
  },
}))

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

const mockAuthService = vi.mocked(authService)

const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

const renderLogin = (initialEntry: string | { pathname: string; state?: unknown } = '/login') => {
  const queryClient = createQueryClient()

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/login" element={<LoginView />} />
          <Route path="/dashboard" element={<div>Dashboard route</div>} />
          <Route path="/protected" element={<div>Protected route</div>} />
          <Route path="/setup" element={<div>Setup route</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe('LoginView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuthService.getStatus.mockResolvedValue({ initialized: true })
    vi.mocked(useAuthModule.useAuth).mockReturnValue(buildAuthMock({ login: vi.fn() }))
  })

  it('logs in successfully and redirects to the requested route', async () => {
    const user = userEvent.setup()
    const login = vi.fn().mockResolvedValue(undefined)

    vi.mocked(useAuthModule.useAuth).mockReturnValue(buildAuthMock({ login }))

    renderLogin({
      pathname: '/login',
      state: { from: { pathname: '/protected' } },
    })

    await user.type(await screen.findByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'super-secret')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(login).toHaveBeenCalledWith('admin', 'super-secret')
    })

    expect(await screen.findByText('Protected route')).toBeInTheDocument()
  })

  it('validates required fields before submitting', async () => {
    const user = userEvent.setup()
    const login = vi.fn()

    vi.mocked(useAuthModule.useAuth).mockReturnValue(buildAuthMock({ login }))

    renderLogin()

    await user.click(await screen.findByRole('button', { name: /sign in/i }))

    expect(await screen.findByText('Username is required')).toBeInTheDocument()
    expect(await screen.findByText('Password is required')).toBeInTheDocument()
    expect(login).not.toHaveBeenCalled()
  })

  it('shows the API error message when login fails', async () => {
    const user = userEvent.setup()
    const login = vi.fn().mockRejectedValue({
      response: {
        data: {
          error: {
            message: 'Invalid username or password',
          },
        },
      },
    })

    vi.mocked(useAuthModule.useAuth).mockReturnValue(buildAuthMock({ login }))

    renderLogin()

    await user.type(await screen.findByLabelText(/username/i), 'admin')
    await user.type(screen.getByLabelText(/password/i), 'wrong-password')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    expect(await screen.findByText('Invalid username or password')).toBeInTheDocument()
    expect(login).toHaveBeenCalledWith('admin', 'wrong-password')
  })
})
