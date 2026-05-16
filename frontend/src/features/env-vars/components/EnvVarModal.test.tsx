import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EnvVarModal } from './EnvVarModal'
import type { EnvVar } from '../services/envService'

const baseFormData = {
  key: 'FLOW_ENABLED',
  value: 'true',
  type: 'boolean' as EnvVar['type'],
  description: 'Controls flow startup',
}

const renderModal = (overrides: Partial<React.ComponentProps<typeof EnvVarModal>> = {}) => {
  const props: React.ComponentProps<typeof EnvVarModal> = {
    formData: baseFormData,
    setFormData: vi.fn(),
    onCancel: vi.fn(),
    onSubmit: vi.fn((event) => event.preventDefault()),
    editing: true,
    isPending: false,
    ...overrides,
  }

  const view = render(<EnvVarModal {...props} />)
  return { ...view, props }
}

describe('EnvVarModal polish', () => {
  beforeEach(() => {
    document.body.style.overflow = ''
    document.documentElement.style.overflow = ''
  })

  afterEach(() => {
    document.body.style.overflow = ''
    document.documentElement.style.overflow = ''
    vi.clearAllMocks()
  })

  it('uses English action copy and closes when the backdrop is clicked', async () => {
    const actor = userEvent.setup()
    const { props } = renderModal()

    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument()

    await actor.click(document.querySelector('.modal-overlay') as HTMLElement)
    expect(props.onCancel).toHaveBeenCalledTimes(1)
  })

  it('locks document scroll while mounted and restores the previous values on unmount', () => {
    document.body.style.overflow = 'auto'
    document.documentElement.style.overflow = 'auto'

    const { unmount } = renderModal()

    expect(document.body.style.overflow).toBe('hidden')
    expect(document.documentElement.style.overflow).toBe('hidden')

    unmount()

    expect(document.body.style.overflow).toBe('auto')
    expect(document.documentElement.style.overflow).toBe('auto')
  })

  it('keeps the value field height stable across type switches', () => {
    renderModal()

    const valueField = screen.getByTestId('env-var-value-field')
    expect(valueField).toHaveClass('min-h-12')
    expect(valueField).toHaveClass('transition-all')
    expect(valueField).toHaveClass('duration-150')
  })

  it('renders the boolean value toggle as an accessible medium control with a status badge', () => {
    renderModal()

    expect(screen.getByRole('checkbox', { name: /boolean value/i })).toHaveClass('toggle-md')
    expect(screen.getByText('true')).toHaveClass('badge')
  })
})
