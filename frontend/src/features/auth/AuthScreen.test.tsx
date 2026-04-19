import { useState } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { AuthScreen } from './AuthScreen'
import type { AuthMode } from '../../common/types'

function AuthScreenHarness({
  onSubmit = vi.fn(),
  busy = false,
}: {
  onSubmit?: (username: string, password: string) => void
  busy?: boolean
}) {
  const [mode, setMode] = useState<AuthMode>('login')

  return <AuthScreen mode={mode} message="" busy={busy} onModeChange={setMode} onSubmit={onSubmit} />
}

describe('AuthScreen', () => {
  it('submits the username and password entered by the user', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()

    render(<AuthScreenHarness onSubmit={onSubmit} />)

    await user.type(screen.getByLabelText(/Username/i), 'admin')
    await user.type(screen.getByLabelText(/Password/i), 'secret')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(onSubmit).toHaveBeenCalledWith('admin', 'secret')
  })

  it('switches to bootstrap mode from the mode toggle', async () => {
    const user = userEvent.setup()

    render(<AuthScreenHarness />)

    await user.click(screen.getByRole('button', { name: 'Bootstrap' }))

    expect(screen.getByRole('heading', { name: 'Create the first administrator' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create account' })).toBeInTheDocument()
  })

  it('disables submission while an auth request is pending', () => {
    render(<AuthScreenHarness busy />)

    expect(screen.getByRole('button', { name: 'Working…' })).toBeDisabled()
  })
})
