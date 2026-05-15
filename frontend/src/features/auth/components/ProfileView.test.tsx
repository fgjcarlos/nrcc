import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ProfileView } from './ProfileView'
import { authService } from '../services/authService'

const mockUser = {
  id: 'user-1',
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
    expect(screen.getByText('viewer')).toBeInTheDocument()
    expect(screen.getByText(/user-1/i)).toBeInTheDocument()
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
      expect(mockAuthService.changePassword).toHaveBeenCalledWith('user-1', 'new-password-123')
    })
  })
})
