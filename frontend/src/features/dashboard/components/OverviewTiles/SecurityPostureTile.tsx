import { Link } from 'react-router-dom';
import { ChevronRight } from 'lucide-react';
import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui';
import { cn } from '@/shared/lib';
import { securityVerdict, type SecurityVerdict } from '../../lib';

/**
 * Security posture section of the Overview sheet (T8).
 *
 * The four surfaces are plain, non-interactive rows: /security has no
 * addressable place per surface (and requireHttps lives on Configuration),
 * so one section link carries the navigation. Only the overall "Exposed"
 * verdict is critical red; a surface that is off reads as neutral text,
 * or amber for adminAuth, whose absence is what exposes the editor.
 */

export interface SecurityPostureSurface {
  id: 'adminAuth' | 'httpNodeAuth' | 'httpStaticAuth' | 'requireHttps';
  enabled: boolean;
}

export interface SecurityPostureTileProps {
  /** Whether each surface is configured; undefined while /config has not answered. */
  adminAuth?: boolean;
  httpNodeAuth?: boolean;
  httpStaticAuth?: boolean;
  requireHttps?: boolean;
}

type SurfaceKey = SecurityPostureSurface['id'];

const SURFACES: ReadonlyArray<SurfaceKey> = ['adminAuth', 'httpNodeAuth', 'httpStaticAuth', 'requireHttps'];

const VERDICT_CHIP: Record<SecurityVerdict, { variant: 'success' | 'warning' | 'danger'; key: string }> = {
  lockedDown: { variant: 'success', key: 'success' },
  partial: { variant: 'warning', key: 'warning' },
  exposed: { variant: 'danger', key: 'danger' },
};

export function SecurityPostureTile(props: SecurityPostureTileProps) {
  const { t } = useT();
  const reported = props.adminAuth !== undefined;
  const values: Record<SurfaceKey, boolean> = {
    adminAuth: Boolean(props.adminAuth),
    httpNodeAuth: Boolean(props.httpNodeAuth),
    httpStaticAuth: Boolean(props.httpStaticAuth),
    requireHttps: Boolean(props.requireHttps),
  };
  const chip = reported ? VERDICT_CHIP[securityVerdict(values)] : undefined;
  const verdictLabel = t(`dashboard:overviewTiles.securityPosture.overall.${chip?.key ?? 'notReported'}`);

  return (
    <section
      data-testid="overview-security-posture-tile"
      className="grid gap-4 py-4 sm:py-6 lg:grid-cols-[16rem_minmax(0,1fr)] lg:gap-6"
      aria-labelledby="overview-security-posture-title"
    >
      <header className="space-y-2">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
          <h2 id="overview-security-posture-title" className="text-sm font-semibold text-ds-text-primary">
            {t('dashboard:overviewTiles.securityPosture.title')}
          </h2>
          <StatusChip
            variant={chip?.variant ?? 'neutral'}
            size="sm"
            ariaLabel={t('dashboard:overviewTiles.securityPosture.statusAria', { state: verdictLabel })}
          >
            {verdictLabel}
          </StatusChip>
        </div>
        <p className="text-xs text-ds-text-secondary">{t('dashboard:overviewTiles.securityPosture.subtitle')}</p>
        <Link
          to="/security"
          className="inline-flex min-h-11 items-center gap-1 text-sm font-medium text-ds-on-tint-primary underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ds-accent-primary sm:min-h-0"
          data-testid="overview-security-posture-link"
        >
          {t('dashboard:overviewTiles.securityPosture.cta')}
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </Link>
      </header>

      <dl className="grid content-start gap-x-6 sm:grid-cols-2">
        {SURFACES.map((key) => {
          const enabled = values[key];
          return (
            <div key={key} className="flex items-baseline justify-between gap-3 border-b border-ds-border-subtle py-2">
              <dt className="text-sm text-ds-text-primary">
                {t(`dashboard:overviewTiles.securityPosture.surfaces.${key}`)}
              </dt>
              <dd
                data-testid={`overview-security-${key}`}
                className={cn(
                  'text-xs font-semibold',
                  !reported
                    ? 'text-ds-text-muted'
                    : enabled
                      ? 'text-ds-on-tint-success'
                      : key === 'adminAuth'
                        ? 'text-ds-on-tint-warning'
                        : 'text-ds-text-secondary',
                )}
              >
                {reported
                  ? t(`dashboard:overviewTiles.securityPosture.states.${enabled ? 'enabled' : 'disabled'}`)
                  : t('dashboard:overviewTiles.securityPosture.overall.notReported')}
              </dd>
            </div>
          );
        })}
      </dl>
    </section>
  );
}
