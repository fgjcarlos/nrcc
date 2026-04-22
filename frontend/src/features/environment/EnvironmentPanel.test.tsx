import { QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '../../api'
import { createTestQueryClient } from '../../test/utils'
import { EnvironmentPanel } from './EnvironmentPanel'

vi.mock('../../api', async () => {
  const actual = await vi.importActual<typeof import('../../api')>('../../api')

  return {
    ...actual,
    api: {
      ...actual.api,
      applyEnvironment: vi.fn(),
    },
  }
})

describe('EnvironmentPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('submits edited and newly added variables', async () => {
    const user = userEvent.setup()
    const onSaved = vi.fn().mockResolvedValue(undefined)
    const onError = vi.fn()

    vi.mocked(api.applyEnvironment).mockResolvedValue({ restartRequired: true, variables: [] })

    render(
      <QueryClientProvider client={createTestQueryClient()}>
        <EnvironmentPanel
          state={{
            restartRequired: false,
            variables: [{ name: 'NODE_ENV', value: 'production' }],
          }}
          loading={false}
          onSaved={onSaved}
          onError={onError}
        />,
      </QueryClientProvider>,
    )

    await user.click(screen.getByRole('button', { name: /add variable/i }))

    const nameInputs = screen.getAllByLabelText('Name')
    fireEvent.change(nameInputs[1], { target: { value: 'API_TOKEN' } })
    fireEvent.change(screen.getAllByLabelText('Value')[1], { target: { value: 'secret-value' } })
    await user.click(screen.getByRole('button', { name: 'Save environment' }))

    await waitFor(() => {
      expect(api.applyEnvironment).toHaveBeenCalledWith([
        { name: 'NODE_ENV', value: 'production' },
        { name: 'API_TOKEN', value: 'secret-value' },
      ])
    })

    expect(onSaved).toHaveBeenCalledOnce()
    expect(onError).not.toHaveBeenCalled()
    expect(await screen.findByText('Managed environment saved. Restart Node-RED to apply the changes.')).toBeInTheDocument()
  })

  it('preserves masked secrets unless explicitly cleared', async () => {
    const user = userEvent.setup()
    const onSaved = vi.fn().mockResolvedValue(undefined)
    const onError = vi.fn()

    vi.mocked(api.applyEnvironment).mockResolvedValue({ restartRequired: true, variables: [] })

    render(
      <QueryClientProvider client={createTestQueryClient()}>
        <EnvironmentPanel
          state={{
            restartRequired: true,
            variables: [{ name: 'API_TOKEN', value: '', secret: true, hasValue: true }],
          }}
          loading={false}
          onSaved={onSaved}
          onError={onError}
        />
      </QueryClientProvider>,
    )

    expect(screen.getByPlaceholderText(/stored secret hidden/i)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Save environment' }))

    await waitFor(() => {
      expect(api.applyEnvironment).toHaveBeenCalledWith([{ name: 'API_TOKEN', value: '', secret: true, hasValue: true }])
    })

    vi.mocked(api.applyEnvironment).mockClear()
    await user.click(screen.getByRole('button', { name: /clear stored secret/i }))
    await user.click(screen.getByRole('button', { name: 'Save environment' }))

    await waitFor(() => {
      expect(api.applyEnvironment).toHaveBeenCalledWith([{ name: 'API_TOKEN', value: '', secret: true, hasValue: false }])
    })
  })
})
