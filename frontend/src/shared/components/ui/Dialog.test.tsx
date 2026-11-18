import { useRef, useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { i18n } from '@/i18n'
import { Dialog, DialogBody, DialogFooter, type DialogProps } from './Dialog'

function renderDialog(props: Partial<DialogProps> = {}) {
  const onClose = vi.fn()
  const view = render(
    <Dialog open title="Edit variable" description="Changes restart Node-RED." onClose={onClose} {...props}>
      <DialogBody>
        <input aria-label="Key" />
      </DialogBody>
      <DialogFooter>
        <button type="button">Cancel</button>
        <button type="button">Save</button>
      </DialogFooter>
    </Dialog>
  )
  return { ...view, onClose }
}

describe('Dialog', () => {
  afterEach(async () => {
    document.body.style.overflow = ''
    document.documentElement.style.overflow = ''
    await i18n.changeLanguage('en')
  })

  it('portals to document.body as a labelled, described modal dialog', () => {
    render(
      <div data-testid="page">
        <Dialog open title="Edit variable" description="Changes restart Node-RED." onClose={vi.fn()} />
      </div>
    )
    const dialog = screen.getByRole('dialog', { name: 'Edit variable' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAccessibleDescription('Changes restart Node-RED.')
    expect(screen.getByRole('heading', { name: 'Edit variable' })).toBeInTheDocument()
    expect(dialog.closest('.modal-overlay')?.parentElement).toBe(document.body)
    expect(screen.getByTestId('page')).not.toContainElement(dialog)
  })

  it('omits aria-describedby when there is no description', () => {
    render(<Dialog open title="Edit variable" onClose={vi.fn()} />)
    expect(screen.getByRole('dialog')).not.toHaveAttribute('aria-describedby')
  })

  it('renders nothing while closed', () => {
    render(<Dialog open={false} title="Edit variable" onClose={vi.fn()} />)
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.querySelector('.modal-overlay')).toBeNull()
  })

  it('labels the close button from the i18n catalog in both locales', async () => {
    const { onClose, unmount } = renderDialog()
    const close = screen.getByRole('button', { name: i18n.t('common:closeDialog') })
    expect(close).toHaveAccessibleName('Close dialog')
    await userEvent.click(close)
    expect(onClose).toHaveBeenCalledTimes(1)
    unmount()

    await i18n.changeLanguage('es')
    renderDialog()
    expect(screen.getByRole('button', { name: 'Cerrar diálogo' })).toBeInTheDocument()
  })

  it('moves initial focus to the first control in the content, not the close button', () => {
    renderDialog()
    expect(screen.getByRole('textbox', { name: 'Key' })).toHaveFocus()
  })

  it('honours an explicit initial focus target', () => {
    function Harness() {
      const saveRef = useRef<HTMLButtonElement>(null)
      return (
        <Dialog open title="Edit variable" onClose={vi.fn()} initialFocusRef={saveRef}>
          <DialogBody><input aria-label="Key" /></DialogBody>
          <DialogFooter><button ref={saveRef} type="button">Save</button></DialogFooter>
        </Dialog>
      )
    }
    render(<Harness />)
    expect(screen.getByRole('button', { name: 'Save' })).toHaveFocus()
  })

  it('keeps Tab and Shift+Tab inside the dialog', async () => {
    const user = userEvent.setup()
    renderDialog()
    const close = screen.getByRole('button', { name: 'Close dialog' })
    const save = screen.getByRole('button', { name: 'Save' })
    save.focus()
    await user.tab()
    expect(close).toHaveFocus()
    await user.tab({ shift: true })
    expect(save).toHaveFocus()
  })

  it('closes on Escape and on a backdrop click, but not on a click inside', async () => {
    const user = userEvent.setup()
    const { onClose } = renderDialog()
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
    await user.click(screen.getByText('Changes restart Node-RED.'))
    expect(onClose).toHaveBeenCalledTimes(1)
    await user.click(document.querySelector('.modal-overlay') as HTMLElement)
    expect(onClose).toHaveBeenCalledTimes(2)
  })

  it('stays open on a backdrop click when closeOnBackdrop is false', async () => {
    const user = userEvent.setup()
    const { onClose } = renderDialog({ closeOnBackdrop: false })
    await user.click(document.querySelector('.modal-overlay') as HTMLElement)
    expect(onClose).not.toHaveBeenCalled()
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('ignores Escape, the backdrop and the close button while not dismissible', async () => {
    const user = userEvent.setup()
    const { onClose } = renderDialog({ dismissible: false })
    await user.keyboard('{Escape}')
    await user.click(document.querySelector('.modal-overlay') as HTMLElement)
    expect(screen.getByRole('button', { name: 'Close dialog' })).toBeDisabled()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('returns focus to the opener when it closes', async () => {
    const user = userEvent.setup()
    function Harness() {
      const [open, setOpen] = useState(false)
      return (
        <>
          <button type="button" onClick={() => setOpen(true)}>Open</button>
          <Dialog open={open} title="Edit variable" onClose={() => setOpen(false)} />
        </>
      )
    }
    render(<Harness />)
    const opener = screen.getByRole('button', { name: 'Open' })
    await user.click(opener)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(opener).toHaveFocus()
  })

  it('locks document scroll while open and restores it on close', () => {
    document.body.style.overflow = 'auto'
    const { unmount } = renderDialog()
    expect(document.body.style.overflow).toBe('hidden')
    expect(document.documentElement.style.overflow).toBe('hidden')
    unmount()
    expect(document.body.style.overflow).toBe('auto')
    expect(document.documentElement.style.overflow).toBe('')
  })
})
