import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { AlertTriangle } from 'lucide-react';
import { StatusChip } from './StatusChip';

describe('StatusChip', () => {
  it('renders the visible label', () => {
    render(<StatusChip variant="success">Saved</StatusChip>);
    expect(screen.getByText('Saved')).toBeInTheDocument();
  });

  it('defaults to neutral variant and md size when omitted', () => {
    render(<StatusChip>Idle</StatusChip>);
    const chip = screen.getByRole('status');
    expect(chip.className).toMatch(/bg-base-300/);
    expect(chip.className).toMatch(/px-3/);
  });

  it('applies the success variant palette', () => {
    render(<StatusChip variant="success">Running</StatusChip>);
    const chip = screen.getByRole('status');
    expect(chip.className).toMatch(/bg-ds-success/);
    expect(chip.className).toMatch(/text-ds-success/);
  });

  it('applies the warning variant palette', () => {
    render(<StatusChip variant="warning">Pending</StatusChip>);
    const chip = screen.getByRole('status');
    expect(chip.className).toMatch(/bg-ds-warning/);
    expect(chip.className).toMatch(/text-ds-warning/);
  });

  it('applies the danger variant palette', () => {
    render(<StatusChip variant="danger">Failed</StatusChip>);
    const chip = screen.getByRole('status');
    expect(chip.className).toMatch(/bg-ds-danger/);
  });

  it('applies the info variant palette', () => {
    render(<StatusChip variant="info">Detected</StatusChip>);
    const chip = screen.getByRole('status');
    expect(chip.className).toMatch(/bg-ds-info/);
  });

  it('honours the size="sm" override', () => {
    render(<StatusChip size="sm">Compact</StatusChip>);
    const chip = screen.getByRole('status');
    expect(chip.className).toMatch(/px-2\.5/);
    expect(chip.className).toMatch(/py-0\.5/);
  });

  it('renders an icon when provided', () => {
    render(
      <StatusChip variant="warning" icon={<AlertTriangle data-testid="chip-icon" />}>
        Restart required
      </StatusChip>,
    );
    expect(screen.getByTestId('chip-icon')).toBeInTheDocument();
    expect(screen.getByText('Restart required')).toBeInTheDocument();
  });

  it('uses aria-label override when provided', () => {
    render(
      <StatusChip variant="success" ariaLabel="Configuration saved successfully">
        Saved
      </StatusChip>,
    );
    const chip = screen.getByRole('status');
    expect(chip).toHaveAttribute('aria-label', 'Configuration saved successfully');
  });

  it('falls back to the visible text for aria-label when no override', () => {
    render(<StatusChip variant="info">Node-RED detected</StatusChip>);
    const chip = screen.getByRole('status');
    expect(chip).toHaveAttribute('aria-label', 'Node-RED detected');
  });

  it('merges custom className with variant classes', () => {
    render(<StatusChip className="ml-2">Custom</StatusChip>);
    const chip = screen.getByRole('status');
    expect(chip.className).toMatch(/ml-2/);
    expect(chip.className).toMatch(/rounded-full/);
  });
});
