import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { SetupView } from './SetupView'
import { authService } from '../services/authService'
import { buildAuthResponseMock } from '../__test-utils__/authMock'
import { i18n } from '@/i18n'

vi.mock('../services/authService', () => ({
  authService: {
    getStatus: vi.fn(),
    setup: vi.fn(),
  },
}))

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

const mockAuthService = vi.mocked(authService)

const renderSetup = () =>
  render(
    <MemoryRouter initialEntries={['/setup']}>
      <Routes>
        <Route path="/setup" element={<SetupView />} />
        <Route path="/dashboard" element={<div>Dashboard route</div>} />
        <Route path="/login" element={<div>Login route</div>} />
      </Routes>
    </MemoryRouter>
  )

describe('SetupView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuthService.getStatus.mockResolvedValue({ initialized: false })
    mockAuthService.setup.mockResolvedValue(buildAuthResponseMock())
  })

  it('creates the initial admin account and redirects to the dashboard', async () => {
    const user = userEvent.setup()

    renderSetup()

    await user.type(await screen.findByLabelText(/^username$/i), 'admin')
    await user.type(screen.getByLabelText(/^password$/i), 'super-secret')
    await user.type(screen.getByLabelText(/confirm password/i), 'super-secret')
    await user.click(screen.getByRole('button', { name: /create account and continue/i }))

    await waitFor(() => {
      expect(mockAuthService.setup).toHaveBeenCalledWith('admin', 'super-secret')
    })

    expect(await screen.findByText('Dashboard route')).toBeInTheDocument()
  })

  it('redirects to login when the system is already initialized', async () => {
    mockAuthService.getStatus.mockResolvedValue({ initialized: true })

    renderSetup()

    expect(await screen.findByText('Login route')).toBeInTheDocument()
    expect(mockAuthService.setup).not.toHaveBeenCalled()
  })
})

const RAW_KEY = /\b(?:auth|common):|\b(?:login|setup)\.[a-zA-Z]/

async function fillAndSubmit() {
  const user = userEvent.setup()
  renderSetup()
  await user.type(await screen.findByLabelText(/^username$/i), 'admin')
  await user.type(screen.getByLabelText(/^password$/i), 'super-secret')
  await user.type(screen.getByLabelText(/confirm password/i), 'super-secret')
  await user.click(screen.getByRole('button', { name: /create account and continue/i }))
  return user
}

describe('SetupView copy, states and controls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuthService.getStatus.mockResolvedValue({ initialized: false })
    mockAuthService.setup.mockResolvedValue(buildAuthResponseMock())
  })

  afterEach(async () => {
    await i18n.changeLanguage('en')
  })

  it.each([
    ['en', 'Create account and continue', /letters, numbers and underscores/],
    ['es', 'Crear cuenta y continuar', /letras, números y guiones bajos/],
  ])('renders no raw i18n keys in %s and localizes validation', async (locale, submit, usernameHint) => {
    await i18n.changeLanguage(locale)
    const user = userEvent.setup()
    renderSetup()

    const button = await screen.findByRole('button', { name: submit })
    expect(screen.getByText('NRCC')).toBeInTheDocument()
    expect(screen.getByText(usernameHint)).toBeInTheDocument()
    expect(document.body.textContent).not.toMatch(RAW_KEY)

    await user.click(button)
    await waitFor(() => expect(screen.getAllByRole('textbox')[0]).toHaveAttribute('aria-invalid', 'true'))
    expect(document.body.textContent).not.toMatch(RAW_KEY)
  })

  it('explains the first admin account and shows the requirements before submitting', async () => {
    renderSetup()

    expect(await screen.findByText(/creates the first administrator account/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^username$/i)).toHaveAccessibleDescription(
      '3 to 32 characters: letters, numbers and underscores.',
    )
    expect(screen.getByLabelText(/^password$/i)).toHaveAccessibleDescription(
      '8 to 72 characters. Common passwords such as password123 are refused.',
    )
  })

  it('wires autocomplete and invalid state on the fields', async () => {
    const user = userEvent.setup()
    renderSetup()

    const username = await screen.findByLabelText(/^username$/i)
    const password = screen.getByLabelText(/^password$/i)
    const confirm = screen.getByLabelText(/confirm password/i)
    expect(username).toHaveAttribute('autocomplete', 'username')
    expect(password).toHaveAttribute('autocomplete', 'new-password')
    expect(confirm).toHaveAttribute('autocomplete', 'new-password')

    await user.type(username, 'ab')
    await user.type(password, 'short')
    await user.type(confirm, 'other')
    await user.click(screen.getByRole('button', { name: /create account and continue/i }))

    await waitFor(() => expect(username).toHaveAttribute('aria-invalid', 'true'))
    expect(username).toHaveAccessibleDescription(/Username must be 3-32 characters/)
    expect(password).toHaveAttribute('aria-invalid', 'true')
    expect(password).toHaveAccessibleDescription(/Password must be at least 8 characters/)
    expect(confirm).toHaveAttribute('aria-invalid', 'true')
    expect(confirm).toHaveAccessibleDescription('Passwords do not match')
    expect(mockAuthService.setup).not.toHaveBeenCalled()
  })

  it('marks the submit button busy and prevents a double submit while pending', async () => {
    mockAuthService.setup.mockImplementation(() => new Promise<never>(() => {}))
    const user = await fillAndSubmit()

    const button = screen.getByRole('button', { name: /creating account/i })
    expect(button).toBeDisabled()
    expect(button).toHaveAttribute('aria-busy', 'true')
    await user.click(button)
    expect(mockAuthService.setup).toHaveBeenCalledTimes(1)
  })

  it('shows setup-already-done inline with a way to sign in', async () => {
    mockAuthService.setup.mockRejectedValue({
      response: { status: 409, data: { error: { code: 'ALREADY_CONFIGURED', message: 'System already configured with users' } } },
    })
    await fillAndSubmit()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('This instance already has an administrator. Sign in with that account.')
    expect(alert.closest('form')).not.toBeNull()
    expect(screen.getByRole('link', { name: 'Go to sign in' })).toHaveAttribute('href', '/login')
  })

  it.each([
    ['weak password', { response: { status: 400, data: { error: { code: 'WEAK_PASSWORD', message: 'password is too common' } } } }, 'The server refused this password: it is too common or longer than 72 characters. Choose another one.'],
    ['rate limit', { response: { status: 429, data: {} } }, 'Too many setup attempts. Wait a few minutes, then try again.'],
    ['network failure', new Error('Network Error'), 'Cannot reach the NRCC server. Check that it is running, then try again.'],
    ['unexpected status', { response: { status: 500, data: {} } }, 'Setup failed: the server answered 500. Try again, or check the NRCC server log.'],
  ])('shows %s inline with role="alert"', async (_name, rejection, copy) => {
    mockAuthService.setup.mockRejectedValue(rejection)
    await fillAndSubmit()

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(copy)
    expect(alert.closest('form')).not.toBeNull()
  })

  it('shows the Spanish setup-already-done copy', async () => {
    await i18n.changeLanguage('es')
    mockAuthService.setup.mockRejectedValue({ response: { status: 409, data: {} } })
    const user = userEvent.setup()
    renderSetup()

    await user.type(await screen.findByLabelText('Usuario'), 'admin')
    await user.type(screen.getByLabelText('Contraseña'), 'super-secret')
    await user.type(screen.getByLabelText('Confirmar contraseña'), 'super-secret')
    await user.click(screen.getByRole('button', { name: 'Crear cuenta y continuar' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Esta instancia ya tiene un administrador. Inicia sesión con esa cuenta.')
  })
})
