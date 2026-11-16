import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Select } from './Select'
import { Textarea } from './Textarea'
import { Toggle } from './Toggle'

describe('Select', () => {
  it('labels the control, renders its options and reports an invalid value', () => {
    render(
      <Select label="Log level" error="Pick a level">
        <option value="info">Info</option>
        <option value="warn">Warning</option>
      </Select>,
    )
    const select = screen.getByLabelText('Log level')
    expect(screen.getByRole('option', { name: 'Warning' })).toBeInTheDocument()
    expect(select).toHaveAttribute('aria-invalid', 'true')
    expect(select).toHaveAccessibleDescription('Pick a level')
  })

  it('passes disabled through and is valid by default', () => {
    render(<Select label="Log level" disabled><option>Info</option></Select>)
    const select = screen.getByLabelText('Log level')
    expect(select).toBeDisabled()
    expect(select).not.toHaveAttribute('aria-invalid')
  })
})

describe('Textarea', () => {
  it('labels the control and reports an invalid value', () => {
    render(<Textarea label="Variables" error="Line 2 has no =" />)
    const textarea = screen.getByLabelText('Variables')
    expect(textarea).toHaveAttribute('aria-invalid', 'true')
    expect(textarea).toHaveAccessibleDescription('Line 2 has no =')
  })

  it('passes the read-only state through', () => {
    render(<Textarea label="Variables" readOnly />)
    expect(screen.getByLabelText('Variables')).toHaveAttribute('readonly')
  })
})

describe('Toggle', () => {
  it('exposes its state with aria-pressed and reports the next value', async () => {
    const onCheckedChange = vi.fn()
    render(<Toggle aria-label="Safe mode" checked={false} onCheckedChange={onCheckedChange} />)
    const toggle = screen.getByRole('button', { name: 'Safe mode' })
    expect(toggle).toHaveAttribute('aria-pressed', 'false')
    await userEvent.click(toggle)
    expect(onCheckedChange).toHaveBeenCalledWith(true)
  })

  it('does not report a change while disabled', async () => {
    const onCheckedChange = vi.fn()
    render(<Toggle aria-label="Safe mode" checked onCheckedChange={onCheckedChange} disabled />)
    const toggle = screen.getByRole('button', { name: 'Safe mode' })
    expect(toggle).toHaveAttribute('aria-pressed', 'true')
    expect(toggle).toBeDisabled()
    await userEvent.click(toggle)
    expect(onCheckedChange).not.toHaveBeenCalled()
  })
})
