import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { AuthScreen } from './AuthScreen'
import type { AuthMode } from '../../common/types'

function AuthScreenHarness({
  onSubmit = vi.fn(),
  busy = false,
  mode = 'login' as AuthMode,
}: {
  onSubmit?: (username: string, password: string) => void
  busy?: boolean
  mode?: AuthMode
}) {
  return <AuthScreen mode={mode} message="" busy={busy} onSubmit={onSubmit} />
}

describe('AuthScreen', () => {
  it('submits the username and password entered by the user', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    render(<AuthScreenHarness onSubmit={onSubmit} />)

    await user.type(screen.getByLabelText(/Username/i), 'admin')
    await user.type(screen.getByLabelText(/Password/i), 'secret')
    await user.click(screen.getByRole('button', { name: 'Sign In' }))

    expect(onSubmit).toHaveBeenCalledWith('admin', 'secret')
  })

  it('disables submission while an auth request is pending', () => {
    render(<AuthScreenHarness busy />)

    expect(screen.getByRole('button', { name: 'Working…' })).toBeDisabled()
  })

  it('renders "Initial Setup" kicker and "Create Administrator Account" heading in register mode', () => {
    render(<AuthScreenHarness mode="register" />)

    expect(screen.getByText('Initial Setup')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Create Administrator Account' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create Administrator' })).toBeInTheDocument()
  })

  it('does not render "Initial Setup" kicker in login mode', () => {
    render(<AuthScreenHarness mode="login" />)

    expect(screen.queryByText('Initial Setup')).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Sign In' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeInTheDocument()
  })
})
