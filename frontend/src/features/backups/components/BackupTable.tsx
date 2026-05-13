import { ArrowUpDown, ChevronLeft, ChevronRight } from 'lucide-react';
import { cn, formatBytes } from '@/shared/lib';
import type { BackupSummary } from '../services/backupService';
import type { SortOrder } from '../types';

export interface BackupTableProps {
  items: BackupSummary[];
  total: number;
  page: number;
  limit: number;
  sort?: 'date' | 'size' | 'status';
  order?: SortOrder;
  isLoading?: boolean;
  onPageChange: (page: number) => void;
  onSort: (sort: 'date' | 'size' | 'status', order: SortOrder) => void;
}

const typeLabels: Record<BackupSummary['type'], string> = {
  manual: 'Manual',
  auto: 'Auto',
  'pre-restore': 'Pre-restore',
};

const typeStyles: Record<BackupSummary['type'], string> = {
  manual: 'bg-success/15 text-success-content',
  auto: 'bg-info/15 text-info-content',
  'pre-restore': 'bg-warning/15 text-warning-content',
};

function formatBackupDate(createdAt: string): string {
  try {
    return new Date(createdAt).toLocaleString();
  } catch {
    return 'Invalid date';
  }
}

export function BackupTable({
  items,
  total,
  page,
  limit,
  sort = 'date',
  order = 'desc',
  isLoading = false,
  onPageChange,
  onSort,
}: BackupTableProps) {
  const totalPages = Math.ceil(total / limit);
  const hasPrevious = page > 1;
  const hasNext = page < totalPages;

  const handleSortClick = (newSort: 'date' | 'size' | 'status') => {
    const newOrder: SortOrder = sort === newSort && order === 'desc' ? 'asc' : 'desc';
    onSort(newSort, newOrder);
  };

  const SortHeader = ({ column, label }: { column: 'date' | 'size' | 'status'; label: string }) => (
    <th
      className="cursor-pointer hover:bg-muted/50 px-4 py-2 text-left text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
      onClick={() => handleSortClick(column)}
    >
      <div className="flex items-center gap-2">
        {label}
        <ArrowUpDown
          size={14}
          className={cn(
            'opacity-50 transition-opacity',
            sort === column && 'opacity-100'
          )}
        />
      </div>
    </th>
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="flex gap-2">
          <div className="w-3 h-3 rounded-full bg-primary animate-pulse" />
          <div className="w-3 h-3 rounded-full bg-primary animate-pulse delay-100" />
          <div className="w-3 h-3 rounded-full bg-primary animate-pulse delay-200" />
        </div>
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground">No backups yet. Configure the scheduler to start creating backups automatically.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto border border-border rounded-lg">
        <table className="w-full">
          <thead className="bg-muted/50">
            <tr>
              <SortHeader column="date" label="Date Created" />
              <th className="px-4 py-2 text-left text-sm font-medium text-muted-foreground">Name</th>
              <th className="px-4 py-2 text-left text-sm font-medium text-muted-foreground">Type</th>
              <SortHeader column="size" label="Size" />
              <th className="px-4 py-2 text-left text-sm font-medium text-muted-foreground">Files</th>
              <th className="px-4 py-2 text-left text-sm font-medium text-muted-foreground">Triggered By</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {items.map((backup) => (
              <tr key={backup.id} className="hover:bg-muted/30 transition-colors">
                <td className="px-4 py-3 text-sm text-foreground">
                  {formatBackupDate(backup.createdAt)}
                </td>
                <td className="px-4 py-3 text-sm text-foreground">{backup.name}</td>
                <td className="px-4 py-3 text-sm">
                  <span className={cn('inline-block px-2 py-1 rounded text-xs font-medium', typeStyles[backup.type])}>
                    {typeLabels[backup.type]}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm text-foreground">
                  {formatBytes(backup.totalSize)}
                </td>
                <td className="px-4 py-3 text-sm text-foreground">{backup.fileCount}</td>
                <td className="px-4 py-3 text-sm text-foreground">{backup.triggeredBy}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-between px-4 py-2 border border-border rounded-lg">
          <span className="text-sm text-muted-foreground">
            Page {page} of {totalPages}
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => onPageChange(page - 1)}
              disabled={!hasPrevious}
              className="inline-flex items-center gap-1 px-3 py-1 rounded text-sm font-medium hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              aria-label="Previous page"
            >
              <ChevronLeft size={16} />
              Previous
            </button>
            <button
              onClick={() => onPageChange(page + 1)}
              disabled={!hasNext}
              className="inline-flex items-center gap-1 px-3 py-1 rounded text-sm font-medium hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              aria-label="Next page"
            >
              Next
              <ChevronRight size={16} />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
