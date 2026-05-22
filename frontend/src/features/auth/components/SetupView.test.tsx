import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { SetupView } from './SetupView'
import { authService } from '../services/authService'

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
    mockAuthService.setup.mockResolvedValue(undefined)
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
