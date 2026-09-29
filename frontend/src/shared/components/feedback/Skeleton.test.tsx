import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Skeleton } from './Skeleton';

describe('Skeleton', () => {
  it('renders an element marked aria-busy="true" so screen readers announce pending state', () => {
    render(<Skeleton variant="text" />);
    expect(screen.getByRole('status')).toHaveAttribute('aria-busy', 'true');
  });

  it('uses the visually-hidden loading label as accessible name', () => {
    render(<Skeleton variant="text" />);
    expect(screen.getByRole('status')).toHaveAccessibleName('Loading...');
  });

  it('renders the text variant with a single-line block height', () => {
    render(<Skeleton variant="text" />);
    const el = screen.getByRole('status').firstElementChild;
    expect(el?.className).toMatch(/h-3|h-4/);
  });

  it('renders the rect variant with a card-like block height', () => {
    render(<Skeleton variant="rect" />);
    const el = screen.getByRole('status').firstElementChild;
    expect(el?.className).toMatch(/h-24|h-32/);
  });

  it('renders the circle variant with rounded-full utility', () => {
    render(<Skeleton variant="circle" />);
    const el = screen.getByRole('status').firstElementChild;
    expect(el?.className).toMatch(/rounded-full/);
  });

  it('applies prefers-reduced-motion:animate-none so shimmer respects user preference', () => {
    render(<Skeleton variant="text" />);
    const el = screen.getByRole('status').firstElementChild;
    expect(el?.className).toMatch(/motion-reduce:animate-none/);
  });

  it('forwards additional className to the inner block', () => {
    render(<Skeleton variant="rect" className="w-64" />);
    const el = screen.getByRole('status').firstElementChild;
    expect(el?.className).toMatch(/w-64/);
  });
});