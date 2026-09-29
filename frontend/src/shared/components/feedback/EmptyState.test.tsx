import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Inbox } from 'lucide-react';
import { EmptyState } from './EmptyState';

describe('EmptyState', () => {
  it('renders the title as the heading level-3 landmark', () => {
    render(<EmptyState title="No backups yet" />);
    expect(screen.getByRole('heading', { level: 3, name: 'No backups yet' })).toBeInTheDocument();
  });

  it('renders an accessible status role so screen readers announce the empty state', () => {
    render(<EmptyState title="No backups yet" />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('renders a description when provided', () => {
    render(<EmptyState title="No backups yet" description="Schedule one to start." />);
    expect(screen.getByText('Schedule one to start.')).toBeInTheDocument();
  });

  it('renders a CTA button when actionLabel and onAction are provided', () => {
    const onAction = vi.fn();
    render(<EmptyState title="No backups yet" actionLabel="Schedule backup" onAction={onAction} />);
    const button = screen.getByRole('button', { name: 'Schedule backup' });
    expect(button).toBeInTheDocument();
    button.click();
    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it('does not render a CTA when actionLabel is omitted', () => {
    render(<EmptyState title="No backups yet" />);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('marks the icon as aria-hidden so it does not duplicate the announcement', () => {
    render(<EmptyState title="No backups yet" icon={<Inbox data-testid="empty-icon" />} />);
    const icon = screen.getByTestId('empty-icon');
    expect(icon).toHaveAttribute('aria-hidden', 'true');
  });
});