import { cn } from '@/shared/lib';
import { StatusChip } from '@/shared/components/ui';
import { EmptyState } from '@/shared/components/feedback';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import type { BackupEvent } from '../services/backupService';
import { useT } from '@/i18n';

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
  const { t } = useT();
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
                className={cn('border-b border-ds-border-subtle py-2 text-sm', event.status === 'error' && 'border-ds-danger/35')}
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="font-medium text-ds-text-primary">
                    {event.backupName || event.message || t('backups:history.event')}
                  </div>
                  {event.status === 'error' && (
                    <StatusChip variant="danger" size="sm">
                      {t('backups:history.failed')}
                    </StatusChip>
                  )}
                </div>
                {event.error && (
                  <div className="mt-1 text-xs text-ds-on-tint-error">{event.error}</div>
                )}
                {(event.prunedCount ?? 0) > 0 && (
                  <div className="mt-1 text-xs text-ds-text-secondary">
                    {t('backups:history.pruned', { count: event.prunedCount })}
                  </div>
                )}
              </div>
            ))}
          </div>

          {showPagination && (
            <div className="flex items-center justify-between gap-3 pt-2 text-xs text-ds-text-secondary">
              <span className="tabular-nums">
                {t('backups:history.pageOfTotal', { page, pages: totalPages, total })}
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => onPageChange(Math.max(1, page - 1))}
                  disabled={page === 1}
                  className="action-btn-ghost min-h-9 min-w-9 justify-center px-0"
                  aria-label={t('common:previousPage')}
                >
                  <ChevronLeft className="h-4 w-4" aria-hidden="true" />
                </button>
                <button
                  onClick={() => onPageChange(Math.min(totalPages, page + 1))}
                  disabled={page === totalPages}
                  className="action-btn-ghost min-h-9 min-w-9 justify-center px-0"
                  aria-label={t('common:nextPage')}
                >
                  <ChevronRight className="h-4 w-4" aria-hidden="true" />
                </button>
              </div>
            </div>
          )}
        </>
      ) : (
        <EmptyState title={t('backups:history.empty')} />
      )}
    </div>
  );
}
