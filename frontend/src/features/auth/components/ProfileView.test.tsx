import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ProfileView } from './ProfileView'
import { authService } from '../services/authService'
import { i18n } from '@/i18n'

const mockUser = {
  id: '3f2b8c1e-9a4d-4e6f-8b2a-1c3d5e7f9a0b',
  username: 'operator',
  role: 'viewer' as const,
  createdAt: '2026-05-15T10:00:00Z',
  updatedAt: '2026-05-16T10:00:00Z',
}

vi.mock('../hooks/useAuth', () => ({
  useAuth: () => ({
    user: mockUser,
    isAuthenticated: true,
    isInitialized: true,
    isLoading: false,
  }),
}))

vi.mock('../services/authService', () => ({
  authService: {
    changePassword: vi.fn(),
  },
}))

vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

const mockAuthService = vi.mocked(authService)

describe('ProfileView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the signed-in user account details', () => {
    render(<ProfileView />)

    expect(screen.getByRole('heading', { name: /profile/i })).toBeInTheDocument()
    expect(screen.getByText('operator')).toBeInTheDocument()
    expect(screen.getByText(i18n.t('auth:roles.viewer'))).toBeInTheDocument()
    expect(screen.getByText(mockUser.id)).toBeInTheDocument()
  })

  it('validates password confirmation before calling the API', async () => {
    const user = userEvent.setup()
    render(<ProfileView />)

    await user.type(screen.getByLabelText(/^new password$/i), 'new-password-123')
    await user.type(screen.getByLabelText(/confirm new password/i), 'different-password')
    await user.click(screen.getByRole('button', { name: /update password/i }))

    expect(await screen.findByText(/passwords do not match/i)).toBeInTheDocument()
    expect(mockAuthService.changePassword).not.toHaveBeenCalled()
  })

  it('lets the signed-in user update their own password', async () => {
    const user = userEvent.setup()
    mockAuthService.changePassword.mockResolvedValue(undefined)
    render(<ProfileView />)

    await user.type(screen.getByLabelText(/^new password$/i), 'new-password-123')
    await user.type(screen.getByLabelText(/confirm new password/i), 'new-password-123')
    await user.click(screen.getByRole('button', { name: /update password/i }))

    await waitFor(() => {
      expect(mockAuthService.changePassword).toHaveBeenCalledWith(mockUser.id, 'new-password-123')
    })
  })

  it('renders the user id as a mono designator that breaks only after hyphens', () => {
    render(<ProfileView />)

    const id = screen.getByText(mockUser.id)
    expect(id.tagName).toBe('DD')
    expect(id).not.toHaveClass('break-all')
    const breaks = id.querySelectorAll('wbr')
    expect(breaks).toHaveLength(4)
    breaks.forEach((wbr) => expect(wbr.previousSibling?.textContent?.endsWith('-')).toBe(true))
  })

  it('shows the password requirements up front and the error inline', async () => {
    const user = userEvent.setup()
    render(<ProfileView />)

    const newPassword = screen.getByLabelText(/^new password$/i)
    expect(newPassword).toHaveAccessibleDescription(i18n.t('auth:setup.passwordHint'))

    await user.type(newPassword, 'short')
    await user.type(screen.getByLabelText(/confirm new password/i), 'short')
    await user.click(screen.getByRole('button', { name: /update password/i }))

    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toBe(i18n.t('auth:setup.errors.passwordLength'))
    expect(within(alert.closest('form') as HTMLElement).getByRole('alert')).toBe(alert)
    expect(mockAuthService.changePassword).not.toHaveBeenCalled()
  })
})
