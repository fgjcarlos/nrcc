import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Input } from './Input'

describe('Input', () => {
  it('is valid by default and keeps the shared control boundary', () => {
    render(<Input label="Port" />)
    const input = screen.getByLabelText('Port')
    expect(input).not.toHaveAttribute('aria-invalid')
    expect(input).not.toHaveAttribute('aria-describedby')
    // The light-theme regression (L11): border-none hid the control edge.
    expect(input.className).not.toMatch(/(^|\s)border-(none|0)(\s|$)/)
  })

  it('marks an invalid field and associates its error message', () => {
    render(<Input label="Port" error="Port must be a number" />)
    const input = screen.getByLabelText('Port')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAccessibleDescription('Port must be a number')
  })

  it('keeps a caller description alongside the error message', () => {
    render(
      <>
        <p id="port-hint">1 to 65535</p>
        <Input label="Port" aria-describedby="port-hint" error="Required" />
      </>,
    )
    expect(screen.getByLabelText('Port')).toHaveAccessibleDescription('1 to 65535 Required')
  })

  it('marks a boolean error invalid without rendering a message', () => {
    render(<Input label="Port" error />)
    const input = screen.getByLabelText('Port')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).not.toHaveAttribute('aria-describedby')
  })

  it('passes the disabled and read-only states through', () => {
    render(
      <>
        <Input label="Host" disabled />
        <Input label="Port" readOnly />
      </>,
    )
    expect(screen.getByLabelText('Host')).toBeDisabled()
    expect(screen.getByLabelText('Port')).toHaveAttribute('readonly')
  })

  it('labels the field in the 12px secondary label style shared with FormFields', () => {
    render(<Input label="Port" />)
    const label = screen.getByText('Port')
    expect(label).toHaveClass('text-xs', 'text-ds-text-secondary')
    expect(label).not.toHaveClass('text-label')
  })
})
