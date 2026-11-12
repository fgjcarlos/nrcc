import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Input } from './Input'

describe('Input focus visibility (#889)', () => {
  it('replaces the removed outline with a solid focus-visible outline', () => {
    render(<Input label="Port" />)
    const input = screen.getByLabelText('Port')
    // The glow alone is a low-contrast tint; keyboard focus needs a solid indicator.
    expect(input).toHaveClass('focus-visible:outline-2')
    expect(input).toHaveClass('focus-visible:outline-offset-2')
    expect(input.className).toMatch(/focus-visible:outline-ds-accent-primary/)
  })

  it('keeps the existing error ring when error is set', () => {
    render(<Input label="Port" error />)
    expect(screen.getByLabelText('Port')).toHaveClass('ring-1', 'ring-error')
  })
})
