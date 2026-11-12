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

  describe('keyboard behaviour (#889)', () => {
    it('moves initial focus into the dialog on open', () => {
      renderModal({ editing: false, formData: { ...baseFormData, key: '', type: 'string', value: '' } })
      expect(screen.getByRole('dialog').contains(document.activeElement)).toBe(true)
      expect(screen.getByPlaceholderText('MY_VARIABLE')).toHaveFocus()
    })

    it('Escape cancels the dialog', async () => {
      const actor = userEvent.setup()
      const { props } = renderModal()
      await actor.keyboard('{Escape}')
      expect(props.onCancel).toHaveBeenCalledTimes(1)
      expect(props.onSubmit).not.toHaveBeenCalled()
    })

    it('keeps Tab and Shift+Tab inside the dialog', async () => {
      const actor = userEvent.setup()
      renderModal()
      const dialog = screen.getByRole('dialog')
      const cancel = screen.getByRole('button', { name: /cancel/i })
      const save = screen.getByRole('button', { name: /save/i })
      save.focus()
      await actor.tab()
      expect(dialog.contains(document.activeElement)).toBe(true)
      expect(document.activeElement).not.toBe(save)
      await actor.tab({ shift: true })
      expect(save).toHaveFocus()
      // Shift+Tab from the first focusable control wraps to the last one.
      const first = dialog.querySelector<HTMLElement>('select, input:not([disabled]), button:not([disabled])')
      first?.focus()
      await actor.tab({ shift: true })
      expect(save).toHaveFocus()
      expect(cancel).not.toHaveFocus()
    })

    it('returns focus to the previously focused control on close', () => {
      const trigger = document.createElement('button')
      trigger.textContent = 'New variable'
      document.body.appendChild(trigger)
      trigger.focus()
      const { unmount } = renderModal()
      expect(trigger).not.toHaveFocus()
      unmount()
      expect(trigger).toHaveFocus()
      trigger.remove()
    })

    it('gives every text-entry control a visible focus indicator instead of a bare outline-none', () => {
      for (const type of ['string', 'number', 'secret'] as const) {
        const { unmount } = renderModal({ formData: { ...baseFormData, type, value: '' } })
        const inputs = screen.getByRole('dialog').querySelectorAll('input, select')
        inputs.forEach((input) => {
          if (/(^|\s)focus:outline-none(\s|$)/.test(input.className)) {
            expect(input.className).toMatch(/focus(-visible)?:ring-2/)
          }
        })
        unmount()
      }
    })
  })
})
