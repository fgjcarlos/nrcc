import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { LoadingBoundary } from './LoadingBoundary';

describe('LoadingBoundary', () => {
  it('renders the loading skeleton when state is "pending"', () => {
    render(
      <LoadingBoundary state={{ status: 'pending' }}>
        <div>children</div>
      </LoadingBoundary>
    );
    expect(screen.getByRole('status', { name: 'Loading...' })).toBeInTheDocument();
    expect(screen.queryByText('children')).not.toBeInTheDocument();
  });

  it('renders the error alert when state is "error"', () => {
    const onRetry = vi.fn();
    render(
      <LoadingBoundary
        state={{ status: 'error', error: new Error('boom'), onRetry }}
      >
        <div>children</div>
      </LoadingBoundary>
    );
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.queryByText('children')).not.toBeInTheDocument();
    screen.getByRole('button', { name: 'Try again' }).click();
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('renders the empty state when state is "empty"', () => {
    render(
      <LoadingBoundary state={{ status: 'empty', emptyTitle: 'Nothing here' }}>
        <div>children</div>
      </LoadingBoundary>
    );
    expect(screen.getByRole('heading', { level: 3, name: 'Nothing here' })).toBeInTheDocument();
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.queryByText('children')).not.toBeInTheDocument();
  });

  it('renders the empty state with CTA when emptyActionLabel is provided', () => {
    const onAction = vi.fn();
    render(
      <LoadingBoundary
        state={{
          status: 'empty',
          emptyTitle: 'Nothing here',
          emptyActionLabel: 'Add item',
          onEmptyAction: onAction,
        }}
      >
        <div>children</div>
      </LoadingBoundary>
    );
    screen.getByRole('button', { name: 'Add item' }).click();
    expect(onAction).toHaveBeenCalledTimes(1);
  });

  it('renders children when state is "success"', () => {
    render(
      <LoadingBoundary state={{ status: 'success' }}>
        <div>children content</div>
      </LoadingBoundary>
    );
    expect(screen.getByText('children content')).toBeInTheDocument();
  });

  it('exposes a useEmptyState helper that returns true when data is an empty array', async () => {
    const { useEmptyState } = await import('./loadingBoundaryHelpers');
    expect(useEmptyState([])).toBe(true);
    expect(useEmptyState(null)).toBe(false);
    expect(useEmptyState(undefined)).toBe(false);
    expect(useEmptyState([{ id: 1 }])).toBe(false);
  });
});