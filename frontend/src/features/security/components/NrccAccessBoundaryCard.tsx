import { Link } from 'react-router-dom';
import { ChevronRight, ShieldCheck, ShieldOff, Users } from 'lucide-react';
import { StatusChip } from '@/shared/components/ui';
import { useT } from '@/i18n';
import type { User } from '@/features/auth/services/authService';

/**
 * NrccAccessBoundaryCard — issue #766 slice E
 *
 * First of the four security-surface boundary cards. NRCC operator
 * access (which user has access, what role) is managed elsewhere at
 * /settings/users. This card is read-only: it consumes the user list
 * that SecurityView already fetches via useUsersData and surfaces
 * the count + role distribution plus a deep-link CTA.
 *
 * Pure presentational component; no React Query subscription here
 * (slice C's read-only pattern — see the d023f39 commit log).
 */

export interface NrccAccessBoundaryCardProps {
  /** Pulled from /api/auth/users by SecurityView's parent query. */
  users?: User[];
}

function countRoles(users: User[] | undefined): {
  admins: number;
  viewers: number;
  total: number;
} {
  if (!users) {
    return { admins: 0, viewers: 0, total: 0 };
  }
  let admins = 0;
  let viewers = 0;
  for (const user of users) {
    if (user.role === 'admin') admins += 1;
    else if (user.role === 'viewer') viewers += 1;
  }
  return { admins, viewers, total: users.length };
}

export function NrccAccessBoundaryCard({ users }: NrccAccessBoundaryCardProps) {
  const { t } = useT();
  const { admins, viewers, total } = countRoles(users);
  const variant: 'success' | 'warning' | 'neutral' = total === 0
    ? 'warning'
    : admins === 0
      ? 'warning'
      : 'success';

  return (
    <article
      data-testid="boundary-nrcc-access"
      aria-labelledby="boundary-nrcc-access-title"
      className="card surface-card border border-border p-6"
    >
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          {variant === 'success' ? (
            <ShieldCheck className="h-6 w-6 text-success" aria-hidden="true" />
          ) : (
            <ShieldOff className="h-6 w-6 text-warning" aria-hidden="true" />
          )}
          <div className="min-w-0">
            <h2
              id="boundary-nrcc-access-title"
              className="text-base font-semibold text-base-content"
            >
              {t('security:nrccAccessBoundary.title')}
            </h2>
            <p className="mt-0.5 text-xs text-base-content/65">
              {t('security:nrccAccessBoundary.subtitle')}
            </p>
          </div>
        </div>
        <StatusChip
          variant={variant}
          size="sm"
          ariaLabel={t('security:nrccAccessBoundary.statusAria', {
            state: total > 0 ? 'configured' : 'empty',
          })}
        >
          {total > 0
            ? t('security:nrccAccessBoundary.statusOn', { count: total })
            : t('security:nrccAccessBoundary.statusOff')}
        </StatusChip>
      </header>

      <div
        className="mt-5 grid grid-cols-1 gap-3 md:grid-cols-2"
        data-testid="boundary-nrcc-access-summary"
      >
        <div className="rounded-xl border border-border/60 bg-base-200/30 px-3 py-3">
          <div className="flex items-center gap-1.5 text-base-content/45 uppercase tracking-[0.18em] text-xs">
            <Users className="h-3 w-3" aria-hidden="true" />
            {t('security:nrccAccessBoundary.total')}
          </div>
          <div
            className="mt-1.5 text-base font-semibold text-base-content"
            data-testid="boundary-nrcc-access-total"
          >
            {total}
          </div>
          <p className="mt-0.5 text-base-content/55">
            {t('security:nrccAccessBoundary.totalHint')}
          </p>
        </div>
        <div className="rounded-xl border border-border/60 bg-base-200/30 px-3 py-3">
          <div className="text-base-content/45 uppercase tracking-[0.18em] text-xs">
            {t('security:nrccAccessBoundary.roleDistribution')}
          </div>
          <div
            className="mt-1.5 text-base font-semibold text-base-content"
            data-testid="boundary-nrcc-access-roles"
          >
            {admins} <span className="text-base-content/55">{t('security:nrccAccessBoundary.admins')}</span>
            {' · '}
            {viewers} <span className="text-base-content/55">{t('security:nrccAccessBoundary.viewers')}</span>
          </div>
          <p className="mt-0.5 text-base-content/55">
            {t('security:nrccAccessBoundary.roleDistributionHint')}
          </p>
        </div>
      </div>

      <Link
        to="/settings/users"
        className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-accent transition-colors hover:text-accent-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-1 focus-visible:ring-offset-base-100"
        data-testid="boundary-nrcc-access-link"
      >
        {t('security:nrccAccessBoundary.cta')}
        <ChevronRight className="h-4 w-4" aria-hidden="true" />
      </Link>
    </article>
  );
}
