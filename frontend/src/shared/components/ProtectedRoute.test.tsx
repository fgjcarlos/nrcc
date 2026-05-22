import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ProtectedRoute } from './ProtectedRoute'
import * as useAuthModule from 'features/auth/hooks/useAuth'
import { buildAuthMock, buildUserMock } from 'features/auth/__test-utils__/authMock'

vi.mock('features/auth/hooks/useAuth', () => ({
  useAuth: vi.fn(),
}))

const renderProtectedRoute = (requiredRole?: 'admin' | 'viewer') =>
  render(
    <MemoryRouter initialEntries={['/admin']}>
      <Routes>
        <Route
          path="/admin"
          element={
            <ProtectedRoute requiredRole={requiredRole}>
              <div>Protected content</div>
            </ProtectedRoute>
          }
        />
        <Route path="/login" element={<div>Login route</div>} />
        <Route path="/dashboard" element={<div>Dashboard route</div>} />
      </Routes>
    </MemoryRouter>
  )

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows a loading spinner while auth is loading', () => {
    vi.mocked(useAuthModule.useAuth).mockReturnValue(
      buildAuthMock({ isLoading: true })
    )

    const { container } = renderProtectedRoute()

    expect(container.querySelector('.animate-spin')).toBeInTheDocument()
    expect(screen.queryByText('Protected content')).not.toBeInTheDocument()
  })

  it('redirects unauthenticated users to login', async () => {
    vi.mocked(useAuthModule.useAuth).mockReturnValue(
      buildAuthMock({ isInitialized: true })
    )

    renderProtectedRoute()

    expect(await screen.findByText('Login route')).toBeInTheDocument()
  })

  it('redirects users without the required role to the dashboard', async () => {
    vi.mocked(useAuthModule.useAuth).mockReturnValue(
      buildAuthMock({
        isAuthenticated: true,
        isInitialized: true,
        user: buildUserMock({ id: 'viewer-1', username: 'viewer', role: 'viewer' }),
      })
    )

    renderProtectedRoute('admin')

    expect(await screen.findByText('Dashboard route')).toBeInTheDocument()
  })

  it('renders the child content for authorized users', () => {
    vi.mocked(useAuthModule.useAuth).mockReturnValue(
      buildAuthMock({
        isAuthenticated: true,
        isInitialized: true,
        user: buildUserMock({ id: 'admin-1', username: 'admin', role: 'admin' }),
      })
    )

    renderProtectedRoute('admin')

    expect(screen.getByText('Protected content')).toBeInTheDocument()
  })
})
