import { Link } from 'react-router-dom';
import { ChevronRight } from 'lucide-react';
import { StatusChip } from '@/shared/components/ui';
import { useT } from '@/i18n';
import type { User } from '@/features/auth/services/authService';
import { SecuritySurface } from './SecuritySurface';

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

  const title = t('security:nrccAccessBoundary.title');
  const statusLabel = total > 0
    ? t('security:nrccAccessBoundary.statusOn', { count: total })
    : t('security:nrccAccessBoundary.statusOff');

  return (
    <SecuritySurface
      anchor="nrcc-access"
      testId="boundary-nrcc-access"
      title={title}
      summary={t('security:nrccAccessBoundary.subtitle')}
      status={
        <StatusChip
          variant={variant}
          size="sm"
          ariaLabel={t('security:nrccAccessBoundary.statusAria', { state: statusLabel })}
        >
          {statusLabel}
        </StatusChip>
      }
      details={
        <dl
          className="flex flex-wrap gap-x-4 text-xs text-ds-text-secondary"
          data-testid="boundary-nrcc-access-summary"
        >
          <div className="flex gap-1">
            <dt>{t('security:nrccAccessBoundary.total')}</dt>
            <dd className="font-semibold tabular-nums text-ds-text-primary" data-testid="boundary-nrcc-access-total">
              {total}
            </dd>
          </div>
          <div className="flex gap-1">
            <dt>{t('security:nrccAccessBoundary.roleDistribution')}</dt>
            <dd className="tabular-nums text-ds-text-primary" data-testid="boundary-nrcc-access-roles">
              {t('security:nrccAccessBoundary.roles', { admins, viewers })}
            </dd>
          </div>
        </dl>
      }
      action={
        <Link
          to="/settings/users"
          className="inline-flex min-h-11 items-center gap-1 text-sm font-medium text-ds-on-tint-primary underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ds-accent-primary sm:min-h-9"
          data-testid="boundary-nrcc-access-link"
        >
          {t('security:nrccAccessBoundary.cta')}
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </Link>
      }
    />
  );
}
