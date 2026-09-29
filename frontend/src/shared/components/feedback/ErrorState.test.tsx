import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AlertTriangle } from 'lucide-react';
import { ErrorState } from './ErrorState';

describe('ErrorState', () => {
  it('renders the title', () => {
    render(<ErrorState title="Could not load backups" />);
    expect(screen.getByRole('heading', { level: 3, name: 'Could not load backups' })).toBeInTheDocument();
  });

  it('renders an alert role so screen readers announce the failure immediately', () => {
    render(<ErrorState title="Could not load backups" />);
    expect(screen.getByRole('alert')).toBeInTheDocument();
  });

  it('renders a description when provided', () => {
    render(<ErrorState title="Could not load backups" description="Network unreachable." />);
    expect(screen.getByText('Network unreachable.')).toBeInTheDocument();
  });

  it('renders a retry button when onRetry is provided', () => {
    const onRetry = vi.fn();
    render(<ErrorState title="Could not load backups" onRetry={onRetry} />);
    const button = screen.getByRole('button', { name: 'Try again' });
    expect(button).toBeInTheDocument();
    button.click();
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('does not render a retry button when onRetry is omitted', () => {
    render(<ErrorState title="Could not load backups" />);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('marks the icon as aria-hidden so it does not duplicate the announcement', () => {
    render(<ErrorState title="Could not load backups" icon={<AlertTriangle data-testid="err-icon" />} />);
    const icon = screen.getByTestId('err-icon');
    expect(icon).toHaveAttribute('aria-hidden', 'true');
  });
});