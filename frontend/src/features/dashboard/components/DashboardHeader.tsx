import { Link } from 'react-router-dom';
import { useT } from '@/i18n';
import type { SummaryAction } from '../lib';

interface DashboardHeaderProps {
  /** One sentence stating the instance's state. */
  summary: string;
  /** At most one next action, chosen by the most urgent condition. */
  action: SummaryAction | null;
}

/**
 * Overview page header (T8): the title, one honest summary sentence and at
 * most one next action, closed by a hairline rule (DESIGN.md page header).
 */
export function DashboardHeader({ summary, action }: DashboardHeaderProps) {
  const { t } = useT();
  return (
    <header className="flex flex-col gap-3 border-b border-ds-border-subtle pb-3 sm:flex-row sm:items-end sm:justify-between">
      <div className="min-w-0 space-y-1">
        <h1 className="text-xl font-bold text-ds-text-primary">{t('dashboard:overview')}</h1>
        <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary" data-testid="overview-summary">
          {summary}
        </p>
      </div>
      {action && (
        <Link
          to={action.to}
          data-testid="overview-summary-action"
          className="action-btn-primary min-h-11 shrink-0 self-start text-sm md:min-h-9 sm:self-auto"
        >
          {t(`dashboard:summary.actions.${action.labelKey}`)}
        </Link>
      )}
    </header>
  );
}
