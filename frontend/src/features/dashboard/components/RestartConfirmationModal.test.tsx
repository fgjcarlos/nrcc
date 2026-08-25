import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { RestartConfirmationModal } from './RestartConfirmationModal'

describe('RestartConfirmationModal (#704 portal)', () => {
  beforeEach(() => {
    document.body.style.overflow = ''
    document.documentElement.style.overflow = ''
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('renders nothing when closed', () => {
    const { container } = render(
      <RestartConfirmationModal isOpen={false} onCancel={vi.fn()} onConfirm={vi.fn()} />
    )
    expect(container).toBeEmptyDOMElement()
    expect(document.querySelector('.modal-overlay')).toBeNull()
  })

  it('portals the dialog into document.body so .modal-overlay is the viewport-level backdrop', () => {
    render(
      <RestartConfirmationModal isOpen={true} onCancel={vi.fn()} onConfirm={vi.fn()} />
    )

    // The overlay must live on document.body, not inside the caller's subtree.
    const overlay = document.body.querySelector('.modal-overlay')
    expect(overlay).not.toBeNull()
    expect(overlay?.parentElement).toBe(document.body)

    // Dialog exposes the right role/labelling for a11y.
    expect(screen.getByRole('dialog', { name: '¿Reiniciar Node-RED?' })).toBeInTheDocument()
  })

  it('action buttons have visible horizontal separation (gap-2)', () => {
    render(
      <RestartConfirmationModal isOpen={true} onCancel={vi.fn()} onConfirm={vi.fn()} />
    )
    const cancelBtn = screen.getByRole('button', { name: /cancelar/i })
    const confirmBtn = screen.getByRole('button', { name: /sí, reiniciar/i })
    expect(cancelBtn.parentElement?.className).toMatch(/gap-2/)
    expect(confirmBtn.parentElement?.className).toMatch(/gap-2/)
  })

  it('clicking the overlay cancels; clicking inside the dialog does not', async () => {
    const actor = userEvent.setup()
    const onCancel = vi.fn()
    const onConfirm = vi.fn()
    render(
      <RestartConfirmationModal isOpen={true} onCancel={onCancel} onConfirm={onConfirm} />
    )

    // Click the backdrop (the .modal-overlay itself), not the dialog inside it.
    await actor.click(document.body.querySelector('.modal-overlay') as HTMLElement)
    expect(onCancel).toHaveBeenCalledTimes(1)
    expect(onConfirm).not.toHaveBeenCalled()

    onCancel.mockReset()

    // Click inside the dialog (the title is inside it) — must NOT cancel.
    await actor.click(screen.getByText('¿Reiniciar Node-RED?'))
    expect(onCancel).not.toHaveBeenCalled()
  })

  it('confirm button fires onConfirm and not onCancel', async () => {
    const actor = userEvent.setup()
    const onCancel = vi.fn()
    const onConfirm = vi.fn()
    render(
      <RestartConfirmationModal isOpen={true} onCancel={onCancel} onConfirm={onConfirm} />
    )
    await actor.click(screen.getByRole('button', { name: /sí, reiniciar/i }))
    expect(onConfirm).toHaveBeenCalledTimes(1)
    expect(onCancel).not.toHaveBeenCalled()
  })
})