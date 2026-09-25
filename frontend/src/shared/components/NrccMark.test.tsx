import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { Antenna } from 'lucide-react';
import { NrccMark } from './NrccMark';

describe('NrccMark', () => {
  it('renders an accessible mark with default "NRCC" label', () => {
    render(<NrccMark />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark).toHaveAttribute('aria-label', 'NRCC');
    expect(mark.tagName).toBe('SPAN');
    expect(mark).toHaveAttribute('role', 'img');
  });

  it('honours a custom ariaLabel', () => {
    render(<NrccMark ariaLabel="Node-RED Control Center logo" />);
    expect(screen.getByTestId('nrcc-mark')).toHaveAttribute(
      'aria-label',
      'Node-RED Control Center logo',
    );
  });

  it('applies size="sm" utility classes', () => {
    render(<NrccMark size="sm" />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/h-9/);
    expect(mark.className).toMatch(/w-9/);
  });

  it('applies size="md" utility classes (default)', () => {
    render(<NrccMark />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/h-11/);
    expect(mark.className).toMatch(/w-11/);
  });

  it('applies size="lg" utility classes', () => {
    render(<NrccMark size="lg" />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/h-14/);
    expect(mark.className).toMatch(/w-14/);
  });

  it('applies the accent tone (Header default)', () => {
    render(<NrccMark tone="accent" />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/text-ds-accent-primary/);
  });

  it('applies the primary tone (Sidebar default)', () => {
    render(<NrccMark tone="primary" />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/text-ds-brand-primary/);
  });

  it('applies the muted tone', () => {
    render(<NrccMark tone="muted" />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/text-base-content\/70/);
  });

  it('renders a button when onClick is provided', () => {
    const handler = vi.fn();
    render(<NrccMark onClick={handler} ariaLabel="Open brand page" />);
    const button = screen.getByTestId('nrcc-mark');
    expect(button.tagName).toBe('BUTTON');
    expect(button).toHaveAttribute('aria-label', 'Open brand page');
    button.click();
    expect(handler).toHaveBeenCalledOnce();
  });

  it('renders the default RadioTower icon', () => {
    render(<NrccMark />);
    // RadioTower lucide icon produces a recognizable svg path.
    expect(screen.getByTestId('nrcc-mark').querySelector('svg')).toBeTruthy();
  });

  it('accepts a custom icon override', () => {
    render(<NrccMark icon={<Antenna data-testid="custom-icon" />} />);
    expect(screen.getByTestId('custom-icon')).toBeInTheDocument();
  });

  it('merges custom className for layout positioning', () => {
    render(<NrccMark className="ml-2 mr-3" />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/ml-2/);
    expect(mark.className).toMatch(/mr-3/);
  });
});
