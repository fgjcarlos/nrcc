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
    // Glyph at a whole multiple of the 16-unit grid.
    expect(mark.className).toContain('[&_svg]:h-4');
  });

  it('applies size="md" utility classes (default)', () => {
    render(<NrccMark />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/h-11/);
    expect(mark.className).toMatch(/w-11/);
    expect(mark.className).toContain('[&_svg]:h-6');
  });

  it('applies size="lg" utility classes', () => {
    render(<NrccMark size="lg" />);
    const mark = screen.getByTestId('nrcc-mark');
    expect(mark.className).toMatch(/h-14/);
    expect(mark.className).toMatch(/w-14/);
    expect(mark.className).toContain('[&_svg]:h-8');
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

  it('draws the closed contact as a decorative square-grid glyph', () => {
    render(<NrccMark />);
    const svg = screen.getByTestId('nrcc-mark').querySelector('svg');
    expect(svg).not.toBeNull();
    // The container carries the accessible name; the glyph stays silent.
    expect(svg).toHaveAttribute('aria-hidden', 'true');
    expect(svg).toHaveAttribute('focusable', 'false');
    expect(svg).toHaveAttribute('viewBox', '0 0 16 16');
  });

  // Reads a numeric attribute from the element or its nearest ancestor.
  const num = (el: Element, attr: string) =>
    Number((el.closest(`[${attr}]`) ?? el).getAttribute(attr));
  const glyph = () => {
    render(<NrccMark />);
    const svg = screen.getByTestId('nrcc-mark').querySelector('svg')!;
    const [left, right] = [...svg.querySelectorAll('circle')].sort(
      (a, b) => num(a, 'cx') - num(b, 'cx'),
    );
    return { svg, left, right, lines: [...svg.querySelectorAll('line')] };
  };

  it('draws two open terminal rings on the centre line, legible at 16px', () => {
    const { svg, left, right } = glyph();
    expect(svg.querySelectorAll('circle')).toHaveLength(2);
    for (const ring of [left, right]) {
      expect(num(ring, 'cy')).toBe(8);
      expect(ring.closest('[stroke]')).toHaveAttribute('stroke', 'currentColor');
      // Open ring: the ground shows through the contact point.
      expect(ring.closest('[fill]')).toHaveAttribute('fill', 'none');
      expect(num(ring, 'stroke-width')).toBeGreaterThanOrEqual(1.5);
    }
    // No verified square any more: the accent was dropped from the mark.
    expect(svg.querySelector('rect')).toBeNull();
  });

  it('runs the conductor from each edge into a terminal ring', () => {
    const { left, right, lines } = glyph();
    const r = num(left, 'r');
    const leftWire = lines.find((l) => num(l, 'x1') === 0)!;
    const rightWire = lines.find((l) => num(l, 'x2') === 16)!;
    expect(leftWire).toBeDefined();
    expect(rightWire).toBeDefined();
    for (const wire of [leftWire, rightWire]) {
      expect(num(wire, 'y1')).toBe(8);
      expect(num(wire, 'y2')).toBe(8);
      expect(num(wire, 'stroke-width')).toBeGreaterThanOrEqual(1.5);
    }
    // Each piece ends under its ring's stroke, never inside the ring.
    expect(num(leftWire, 'x2')).toBe(num(left, 'cx') - r);
    expect(num(rightWire, 'x1')).toBe(num(right, 'cx') + r);
  });

  it('lays the closed blade flat between the terminals, heavier than the conductor', () => {
    const { left, right, lines } = glyph();
    expect(lines).toHaveLength(3);
    const r = num(left, 'r');
    const blade = lines.find((l) => num(l, 'x1') === num(left, 'cx') + r)!;
    expect(blade).toBeDefined();
    expect(num(blade, 'x2')).toBe(num(right, 'cx') - r);
    expect(num(blade, 'y1')).toBe(8);
    expect(num(blade, 'y2')).toBe(8);
    const conductor = lines.find((l) => num(l, 'x1') === 0)!;
    expect(num(blade, 'stroke-width')).toBeGreaterThan(num(conductor, 'stroke-width'));
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
