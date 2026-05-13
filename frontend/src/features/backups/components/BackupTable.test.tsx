import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { BackupTable } from './BackupTable';
import type { BackupSummary } from '../services/backupService';

describe('BackupTable', () => {
  const mockBackups: BackupSummary[] = [
    {
      id: 'b1',
      name: 'backup-1',
      type: 'manual',
      createdAt: '2026-05-11T20:00:00Z',
      triggeredBy: 'System',
      fileCount: 5,
      totalSize: 1024,
    },
    {
      id: 'b2',
      name: 'backup-2',
      type: 'auto',
      createdAt: '2026-05-10T20:00:00Z',
      triggeredBy: 'Scheduler',
      fileCount: 3,
      totalSize: 2048,
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render table with headers', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    expect(screen.getByText(/Date Created/i)).toBeInTheDocument();
    expect(screen.getByText(/Size/i)).toBeInTheDocument();
  });

  it('should render backup items in rows', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    expect(screen.getByText('backup-1')).toBeInTheDocument();
    expect(screen.getByText('backup-2')).toBeInTheDocument();
  });

  it('should emit onSort when column header is clicked', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    const sizeHeader = screen.getByText(/Size/i);
    fireEvent.click(sizeHeader.closest('th') || sizeHeader);

    expect(onSort).toHaveBeenCalledWith('size', 'desc');
  });

  it('should show empty state when no items', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={[]}
        total={0}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    expect(screen.getByText(/No backups yet/i)).toBeInTheDocument();
  });

  it('should show loading state', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={[]}
        total={0}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={true}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    // Check for loading indicator (animated dots)
    const loadingContainer = document.querySelector('.animate-pulse');
    expect(loadingContainer).toBeInTheDocument();
  });

  it('should show pagination controls when total > limit', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    expect(screen.getByText(/Page 1 of 3/i)).toBeInTheDocument();
  });

  it('should not show pagination when total <= limit', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={2}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    // Look for pagination - should not exist if only 1 page
    const paginationText = screen.queryByText(/Page 1 of/i);
    expect(paginationText).not.toBeInTheDocument();
  });

  it('should emit onPageChange when next button clicked', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    const nextButton = screen.getByText('Next');
    fireEvent.click(nextButton);

    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  it('should render backup type badges with correct styles', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    expect(screen.getByText('Manual')).toBeInTheDocument();
    expect(screen.getByText('Auto')).toBeInTheDocument();
  });

  it('should sort ascending when clicking same header twice', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    const { rerender } = render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    const dateHeader = screen.getByText(/Date Created/i);
    fireEvent.click(dateHeader.closest('th') || dateHeader);

    expect(onSort).toHaveBeenCalledWith('date', 'asc');
  });

  it('should handle multiple backup types in table', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    const backupsWithTypes: BackupSummary[] = [
      { ...mockBackups[0], type: 'manual' },
      { ...mockBackups[1], type: 'auto' },
      {
        id: 'b3',
        name: 'backup-3',
        type: 'pre-restore',
        createdAt: '2026-05-09T20:00:00Z',
        triggeredBy: 'Restore',
        fileCount: 10,
        totalSize: 4096,
      },
    ];

    render(
      <BackupTable
        items={backupsWithTypes}
        total={3}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    expect(screen.getByText('Manual')).toBeInTheDocument();
    expect(screen.getByText('Auto')).toBeInTheDocument();
    expect(screen.getByText('Pre-restore')).toBeInTheDocument();
  });

  it('should display file count correctly', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    // Check that file counts are displayed
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('should disable previous button on first page', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={1}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    const prevButton = screen.getByText('Previous');
    expect(prevButton).toBeDisabled();
  });

  it('should disable next button on last page', () => {
    const onPageChange = vi.fn();
    const onSort = vi.fn();

    render(
      <BackupTable
        items={mockBackups}
        total={42}
        page={3}
        limit={20}
        sort="date"
        order="desc"
        isLoading={false}
        onPageChange={onPageChange}
        onSort={onSort}
      />
    );

    const nextButton = screen.getByText('Next');
    expect(nextButton).toBeDisabled();
  });
});
