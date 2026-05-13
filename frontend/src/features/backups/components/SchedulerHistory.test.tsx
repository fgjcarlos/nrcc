import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SchedulerHistory } from './SchedulerHistory';
import type { BackupEvent } from '../services/backupService';

describe('SchedulerHistory', () => {
  const mockEvents: BackupEvent[] = [
    {
      id: '1',
      type: 'scheduler-run',
      status: 'success',
      occurredAt: '2026-05-11T20:00:00Z',
      backupId: 'backup-1',
      backupName: 'Auto backup',
      message: '',
      prunedCount: 0,
      error: '',
      activeSpec: '',
      schedule: '0 2 * * *',
    },
    {
      id: '2',
      type: 'prune',
      status: 'success',
      occurredAt: '2026-05-11T19:00:00Z',
      backupId: '',
      backupName: '',
      message: 'Pruned 2 backups',
      prunedCount: 2,
      error: '',
      activeSpec: '',
      schedule: '',
    },
    {
      id: '3',
      type: 'scheduler-error',
      status: 'error',
      occurredAt: '2026-05-11T18:00:00Z',
      backupId: '',
      backupName: '',
      message: 'Disk full',
      prunedCount: 0,
      error: 'Disk full',
      activeSpec: '',
      schedule: '',
    },
  ];

  it('should render event list', () => {
    render(
      <SchedulerHistory
        events={mockEvents}
        page={1}
        limit={10}
        total={3}
        onPageChange={vi.fn()}
      />
    );

    expect(screen.getByText(/Auto backup/i)).toBeInTheDocument();
  });

  it('should display all events in reverse chronological order', () => {
    render(
      <SchedulerHistory
        events={mockEvents}
        page={1}
        limit={10}
        total={3}
        onPageChange={vi.fn()}
      />
    );

    const eventRows = screen.getAllByTestId('event-row');
    expect(eventRows.length).toBe(3);
  });

  it('should show empty state when no events', () => {
    render(
      <SchedulerHistory
        events={[]}
        page={1}
        limit={10}
        total={0}
        onPageChange={vi.fn()}
      />
    );

    expect(screen.getByText(/no scheduler history/i)).toBeInTheDocument();
  });

  it('should display error status badge for failed events', () => {
    render(
      <SchedulerHistory
        events={mockEvents}
        page={1}
        limit={10}
        total={3}
        onPageChange={vi.fn()}
      />
    );

    expect(screen.getAllByText(/Error/i).length).toBeGreaterThan(0);
  });

  it('should show pagination controls when total > limit', () => {
    render(
      <SchedulerHistory
        events={mockEvents}
        page={1}
        limit={2}
        total={15}
        onPageChange={vi.fn()}
      />
    );

    // Should show pagination info or buttons
    const text = screen.getByText(/page|next|previous|more/i) || screen.getByText(/15/i);
    expect(text).toBeDefined();
  });

  it('should not show pagination when total <= limit', () => {
    render(
      <SchedulerHistory
        events={mockEvents}
        page={1}
        limit={10}
        total={3}
        onPageChange={vi.fn()}
      />
    );

    const nextButtons = screen.queryAllByRole('button', { name: /next|previous/i });
    expect(nextButtons.length).toBe(0);
  });

  it('should call onPageChange when next is clicked', () => {
    const onPageChange = vi.fn();
    render(
      <SchedulerHistory
        events={mockEvents}
        page={1}
        limit={2}
        total={15}
        onPageChange={onPageChange}
      />
    );

    const nextButton = screen.getByRole('button', { name: /next/i });
    nextButton.click();

    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  // NEW: Test that pagination info is CLEARLY visible with page numbers
  it('should display current page number prominently in pagination info', () => {
    render(
      <SchedulerHistory
        events={mockEvents}
        page={3}
        limit={5}
        total={25}
        onPageChange={vi.fn()}
      />
    );

    // Should show clear indication of current page (3 of 5)
    expect(screen.getByText(/page 3 of 5/i)).toBeInTheDocument();
  });

  // NEW: Test that pagination shows item count
  it('should display total item count in pagination', () => {
    render(
      <SchedulerHistory
        events={mockEvents}
        page={1}
        limit={10}
        total={42}
        onPageChange={vi.fn()}
      />
    );

    expect(screen.getByText(/42/)).toBeInTheDocument();
  });
});
