import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { UserMenu } from './UserMenu'

const logout = vi.fn()
const user = {
  id: 'user-1',
  username: 'operator',
  role: 'viewer' as const,
  createdAt: '2026-05-15T10:00:00Z',
}

describe('UserMenu profile navigation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('navigates to the profile page from the user menu', async () => {
    const actor = userEvent.setup()

    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route path="/dashboard" element={<UserMenu user={user} onLogout={logout} />} />
          <Route path="/profile" element={<div>Profile route</div>} />
        </Routes>
      </MemoryRouter>
    )

    await actor.click(screen.getByRole('button', { name: /operator .* open user menu/i }))
    await actor.click(screen.getByRole('menuitem', { name: /profile/i }))

    expect(await screen.findByText('Profile route')).toBeInTheDocument()
  })

  // WCAG 2.5.3 (label in name): the visible text of the trigger has to be
  // part of its accessible name. An author label that drops the rendered
  // initials or the role breaks the axe label-content-name-mismatch rule,
  // so the name is asserted against the whole visible text here.
  it('keeps the visible trigger text inside the accessible name', () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route path="/dashboard" element={<UserMenu user={user} onLogout={logout} />} />
        </Routes>
      </MemoryRouter>
    )

    expect(
      screen.getByRole('button', { name: /^OP operator viewer Open user menu$/i }),
    ).toBeInTheDocument()
  })
})
