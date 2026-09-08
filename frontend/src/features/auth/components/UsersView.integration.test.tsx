import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { http, HttpResponse } from 'msw'
import { UsersView } from './UsersView'
import * as useAuthModule from '../hooks/useAuth'
import { buildAuthMock, buildUserMock } from '../__test-utils__/authMock'
import type { User } from '../services/authService'
import { server } from '@/test/msw/server'
import { mockUser } from '@/test/msw/fixtures'
import { UI_COPY } from '@/shared/constants/uiCopy'

vi.mock('../hooks/useAuth', () => ({ useAuth: vi.fn() }))

const ok = <T,>(data: T) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() })

const users: User[] = [
  mockUser,
  { id: 'user-viewer', username: 'viewer', role: 'viewer', createdAt: '2026-01-02T00:00:00.000Z' },
  { id: 'user-admin-2', username: 'admin-two', role: 'admin', createdAt: '2026-01-03T00:00:00.000Z' },
]

function renderAccessManagement() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <UsersView />
      <Toaster />
    </QueryClientProvider>,
  )
}

describe('UsersView access administration integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(useAuthModule.useAuth).mockReturnValue(
      buildAuthMock({
        isAuthenticated: true,
        isInitialized: true,
        user: buildUserMock(mockUser),
      }),
    )
    server.use(http.get('/api/auth/users', () => ok({ users })))
  })

  it('renders users from the backend envelope and protects the current operator', async () => {
    renderAccessManagement()

    expect(await screen.findAllByText('viewer')).not.toHaveLength(0)
    expect(screen.getByText(UI_COPY.userManagementSafetyNotice)).toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: UI_COPY.delete })[0]).toBeDisabled()

    await userEvent.setup().click(screen.getAllByRole('button', { name: UI_COPY.editUser })[0])

    expect(screen.getByDisplayValue('Admin')).toBeDisabled()
    expect(screen.getByText(UI_COPY.cannotChangeOwnRole)).toBeInTheDocument()
  })

  it('sends create, role-update, and delete requests through the users transport', async () => {
    const requests: Array<{ method: string; path: string; body?: unknown }> = []
    server.use(
      http.post('/api/auth/users', async ({ request }) => {
        requests.push({ method: 'POST', path: new URL(request.url).pathname, body: await request.json() })
        return ok({ id: 'user-new', username: 'new-user', role: 'viewer', createdAt: '2026-01-04T00:00:00.000Z' })
      }),
      http.patch('/api/auth/users/:id', async ({ params, request }) => {
        requests.push({ method: 'PATCH', path: `/api/auth/users/${params.id}`, body: await request.json() })
        return ok({ ...users[2], role: 'viewer' })
      }),
      http.delete('/api/auth/users/:id', ({ params }) => {
        requests.push({ method: 'DELETE', path: `/api/auth/users/${params.id}` })
        return ok({ success: true })
      }),
    )
    const user = userEvent.setup()
    renderAccessManagement()
    await screen.findAllByText('viewer')

    await user.click(screen.getByRole('button', { name: new RegExp(`${UI_COPY.add}.*${UI_COPY.createUser}`) }))
    await user.type(screen.getByRole('textbox'), 'new-user')
    const passwordInput = screen.getAllByDisplayValue('').find(
      (element) => element instanceof HTMLInputElement && element.type === 'password',
    )
    if (!(passwordInput instanceof HTMLInputElement)) {
      throw new Error('Password input is missing')
    }
    await user.type(passwordInput, 'securepass123')
    await user.click(screen.getByRole('button', { name: UI_COPY.createUser }))

    await waitFor(() => {
      expect(requests).toContainEqual({
        method: 'POST',
        path: '/api/auth/users',
        body: { username: 'new-user', password: 'securepass123', role: 'viewer' },
      })
    })

    await user.click(screen.getAllByRole('button', { name: UI_COPY.editUser })[2])
    await user.selectOptions(screen.getByDisplayValue('Admin'), 'viewer')
    await user.click(screen.getByRole('button', { name: UI_COPY.confirm }))

    await waitFor(() => {
      expect(requests).toContainEqual({ method: 'PATCH', path: '/api/auth/users/user-admin-2', body: { role: 'viewer' } })
    })

    await user.click(screen.getAllByRole('button', { name: UI_COPY.delete })[1])
    await user.click(screen.getByRole('button', { name: UI_COPY.confirm }))

    await waitFor(() => {
      expect(requests).toContainEqual({ method: 'DELETE', path: '/api/auth/users/user-viewer' })
    })
  })
})
