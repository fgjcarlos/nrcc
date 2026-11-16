import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Button, type ButtonProps } from './Button'

describe('Button', () => {
  it('defaults to the primary action button at md size', () => {
    render(<Button>Review</Button>)
    const button = screen.getByRole('button', { name: 'Review' })
    expect(button).toHaveClass('action-btn-primary')
    expect(button.className).not.toMatch(/min-h-(8|11)/)
  })

  it.each<[NonNullable<ButtonProps['variant']>, string]>([
    ['primary', 'action-btn-primary'],
    ['solid', 'action-btn-solid'],
    ['secondary', 'action-btn-secondary'],
    ['ghost', 'action-btn-ghost'],
    ['danger', 'action-btn-danger'],
  ])('renders the %s variant with the shared %s class', (variant, actionClass) => {
    render(<Button variant={variant}>Go</Button>)
    const button = screen.getByRole('button', { name: 'Go' })
    expect(button).toHaveClass(actionClass)
    // One system: no daisyUI .btn-* class on the component.
    expect(button.className).not.toMatch(/(^|\s)btn-/)
  })

  it.each([
    ['sm', 'min-h-8'],
    ['lg', 'min-h-11'],
  ] as const)('sizes %s to a target of at least 24px (%s)', (size, minHeight) => {
    render(<Button size={size}>Go</Button>)
    expect(screen.getByRole('button', { name: 'Go' })).toHaveClass(minHeight)
  })

  it('forwards disabled and merges caller classes', () => {
    render(<Button disabled className="mt-2">Go</Button>)
    const button = screen.getByRole('button', { name: 'Go' })
    expect(button).toBeDisabled()
    expect(button).toHaveClass('mt-2', 'action-btn-primary')
  })
})
