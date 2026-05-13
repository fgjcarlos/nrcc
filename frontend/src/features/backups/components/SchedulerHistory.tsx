import { cn } from '@/shared/lib';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import type { BackupEvent } from '../services/backupService';

export interface SchedulerHistoryProps {
  events: BackupEvent[];
  page: number;
  limit: number;
  total: number;
  onPageChange: (page: number) => void;
}

export function SchedulerHistory({
  events,
  page,
  limit,
  total,
  onPageChange,
}: SchedulerHistoryProps) {
  const totalPages = Math.ceil(total / limit);
  const showPagination = total > limit;

  return (
    <div className="space-y-3">
      {events.length > 0 ? (
        <>
          <div className="space-y-2">
            {events.map((event) => (
              <div
                key={event.id}
                data-testid="event-row"
                className={cn(
                  'rounded-2xl border p-3 text-sm',
                  event.status === 'error'
                    ? 'border-error/25 bg-error/8'
                    : 'border-border/50 bg-base-200/20'
                )}
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="font-medium text-base-content">
                    {event.backupName || event.message || 'Event'}
                  </div>
                  {event.status === 'error' && (
                    <span className="inline-block rounded-full bg-error/20 px-2.5 py-0.5 text-xs font-medium text-error">
                      Error
                    </span>
                  )}
                </div>
                {event.error && (
                  <div className="mt-1 text-xs text-error">{event.error}</div>
                )}
                {event.prunedCount > 0 && (
                  <div className="mt-1 text-xs text-base-content/60">
                    Pruned {event.prunedCount} backups
                  </div>
                )}
              </div>
            ))}
          </div>

          {showPagination && (
            <div className="flex items-center justify-between gap-3 pt-2 text-xs text-base-content/60">
              <span>
                Page {page} of {totalPages} ({total} total)
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => onPageChange(Math.max(1, page - 1))}
                  disabled={page === 1}
                  className="action-btn-ghost rounded-lg p-1.5 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-label="Previous page"
                >
                  <ChevronLeft className="h-4 w-4" />
                </button>
                <button
                  onClick={() => onPageChange(Math.min(totalPages, page + 1))}
                  disabled={page === totalPages}
                  className="action-btn-ghost rounded-lg p-1.5 disabled:cursor-not-allowed disabled:opacity-50"
                  aria-label="Next page"
                >
                  <ChevronRight className="h-4 w-4" />
                </button>
              </div>
            </div>
          )}
        </>
      ) : (
        <div className="rounded-2xl border border-dashed border-border bg-base-200/15 p-6 text-center text-sm text-base-content/60">
          No scheduler history yet
        </div>
      )}
    </div>
  );
}
