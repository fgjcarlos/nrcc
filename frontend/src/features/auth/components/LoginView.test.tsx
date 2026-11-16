import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { LoginView } from './LoginView'
import { authService } from '../services/authService'
import * as useAuthModule from '../hooks/useAuth'
import { buildAuthMock } from '../__test-utils__/authMock'
import { i18n } from '@/i18n'

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
        status: 401,
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

    expect(await screen.findByRole('alert')).toHaveTextContent('Wrong username or password.')
    expect(login).toHaveBeenCalledWith('admin', 'wrong-password')
  })
})

const RAW_KEY = /\b(?:auth|common):|\b(?:login|setup)\.[a-zA-Z]/

async function submitWith(login: ReturnType<typeof useAuthModule.useAuth>['login']) {
  const user = userEvent.setup()
  vi.mocked(useAuthModule.useAuth).mockReturnValue(buildAuthMock({ login }))
  renderLogin()
  await user.type(await screen.findByLabelText(/username/i), 'admin')
  await user.type(screen.getByLabelText(/password/i), 'secret-pass')
  await user.click(screen.getByRole('button', { name: /sign in/i }))
  return user
}

describe('LoginView copy, states and controls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuthService.getStatus.mockResolvedValue({ initialized: true })
    vi.mocked(useAuthModule.useAuth).mockReturnValue(buildAuthMock({ login: vi.fn() }))
  })

  afterEach(async () => {
    await i18n.changeLanguage('en')
  })

  it.each([
    ['en', 'Sign in'],
    ['es', 'Iniciar sesión'],
  ])('renders no raw i18n keys in %s', async (locale, submit) => {
    await i18n.changeLanguage(locale)
    const user = userEvent.setup()
    renderLogin()

    const button = await screen.findByRole('button', { name: submit })
    expect(screen.getByText('NRCC')).toBeInTheDocument()
    expect(document.body.textContent).not.toMatch(RAW_KEY)

    // Field validation copy is localized too, never a key.
    await user.click(button)
    expect(await screen.findAllByText(locale === 'es' ? /obligatori/ : /is required/)).toHaveLength(2)
    expect(document.body.textContent).not.toMatch(RAW_KEY)
  })

  it.each([
    ['es', 'Iniciar sesión'],
    ['en', 'Sign in'],
  ])('renders the uninitialized state without raw keys in %s', async (locale) => {
    await i18n.changeLanguage(locale)
    mockAuthService.getStatus.mockResolvedValue({ initialized: false })
    renderLogin()

    expect(await screen.findByRole('link', { name: locale === 'es' ? /crear/i : /create/i })).toHaveAttribute('href', '/setup')
    expect(document.body.textContent).not.toMatch(RAW_KEY)
  })

  it('wires labels, autocomplete and invalid state on the fields', async () => {
    const user = userEvent.setup()
    renderLogin()

    const username = await screen.findByLabelText('Username')
    const password = screen.getByLabelText('Password')
    expect(username).toHaveAttribute('autocomplete', 'username')
    expect(password).toHaveAttribute('autocomplete', 'current-password')

    await user.click(screen.getByRole('button', { name: 'Sign in' }))
    await waitFor(() => expect(username).toHaveAttribute('aria-invalid', 'true'))
    expect(username).toHaveAccessibleDescription('Username is required')
    expect(password).toHaveAttribute('aria-invalid', 'true')
    expect(password).toHaveAccessibleDescription('Password is required')
  })

  it('marks the submit button busy and prevents a double submit while pending', async () => {
    const login = vi.fn(() => new Promise<never>(() => {}))
    const user = await submitWith(login)

    const button = screen.getByRole('button', { name: /signing in/i })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('aria-busy', 'true')
    await user.click(button)
    expect(login).toHaveBeenCalledTimes(1)
  })

  it.each([
    ['wrong credentials', { response: { status: 401, data: { error: { message: 'Invalid username or password' } } } }, 'Wrong username or password.'],
    ['rate limit', { response: { status: 429, data: { error: { message: 'Too many attempts. Please try again later.' } } } }, 'Too many sign-in attempts. Wait a few minutes, then try again.'],
    ['network failure', new Error('Network Error'), 'Cannot reach the NRCC server. Check that it is running, then try again.'],
    ['unexpected status', { response: { status: 500, data: {} } }, 'Sign-in failed: the server answered 500. Try again, or check the NRCC server log.'],
  ])('shows %s inline with role="alert"', async (_name, rejection, copy) => {
    const login = vi.fn().mockRejectedValue(rejection)
    await submitWith(login)

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(copy)
    // Inline, inside the sign-in form.
    expect(alert.closest('form')).not.toBeNull()
  })

  it('shows the Spanish wrong-credentials copy', async () => {
    await i18n.changeLanguage('es')
    const user = userEvent.setup()
    const login = vi.fn().mockRejectedValue({ response: { status: 401, data: {} } })
    vi.mocked(useAuthModule.useAuth).mockReturnValue(buildAuthMock({ login }))
    renderLogin()

    await user.type(await screen.findByLabelText('Usuario'), 'admin')
    await user.type(screen.getByLabelText('Contraseña'), 'secret-pass')
    await user.click(screen.getByRole('button', { name: 'Iniciar sesión' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Usuario o contraseña incorrectos.')
  })
})
