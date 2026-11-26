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
    // The glyph is the tile: it fills the container.
    expect(mark.className).toContain('[&_svg]:h-full');
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

  it('is the tile itself: no container border, ground or radius', () => {
    render(<NrccMark />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).not.toMatch(/(^|\s)border(\s|-)/);
    expect(mark.className).not.toMatch(/(^|\s)bg-/);
    expect(mark.className).not.toMatch(/(^|\s)rounded/);
  });

  it('applies the rail tone on the ink navigation rail', () => {
    render(<NrccMark tone="rail" />);
    expect(screen.getByTestId('nrcc-mark').className).toContain('text-[var(--ds-rail-text)]');
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

  const glyph = () => {
    render(<NrccMark />);
    const svg = screen.getByTestId('nrcc-mark').querySelector('svg')!;
    return { svg, mask: svg.querySelector('mask')! };
  };

  it('draws a decorative glyph on a 32-unit grid', () => {
    const { svg } = glyph();
    // The container carries the accessible name; the glyph stays silent.
    expect(svg).toHaveAttribute('aria-hidden', 'true');
    expect(svg).toHaveAttribute('focusable', 'false');
    expect(svg).toHaveAttribute('viewBox', '0 0 32 32');
  });

  it('fills the whole tile with the tone colour through the knockout mask', () => {
    const { svg, mask } = glyph();
    const tile = svg.querySelector(':scope > rect')!;
    expect(tile).toHaveAttribute('width', '32');
    expect(tile).toHaveAttribute('height', '32');
    expect(tile).toHaveAttribute('fill', 'currentColor');
    expect(tile).toHaveAttribute('mask', `url(#${mask.id})`);
  });

  it('knocks out a 2x2 grid that crosses at the centre', () => {
    const { mask } = glyph();
    const rules = [...mask.querySelectorAll('line')];
    expect(rules).toHaveLength(2);
    const horizontal = rules.find((l) => l.getAttribute('y1') === l.getAttribute('y2'))!;
    const vertical = rules.find((l) => l.getAttribute('x1') === l.getAttribute('x2'))!;
    expect(horizontal).toHaveAttribute('y1', '16');
    expect(vertical).toHaveAttribute('x1', '16');
  });

  it('spells N R / C C as stroke paths, one per cell, with no font dependency', () => {
    const { svg, mask } = glyph();
    expect(svg.querySelector('text')).toBeNull();
    const letters = [...mask.querySelectorAll('path')].map((p) => p.getAttribute('data-letter'));
    expect(letters).toEqual(['N', 'R', 'C1', 'C2']);
  });

  it('gives every instance its own mask id', () => {
    render(
      <>
        <NrccMark />
        <NrccMark />
      </>,
    );
    const ids = screen.getAllByTestId('nrcc-mark').map((m) => m.querySelector('mask')!.id);
    expect(new Set(ids).size).toBe(2);
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
