import { Link } from 'react-router-dom';
import { ShieldCheck, ShieldOff, ChevronRight } from 'lucide-react';
import { useT } from '@/i18n';
import { StatusChip } from '@/shared/components/ui';

/**
 * SecurityPostureTile — issue #766 slice D
 *
 * Operational decision tile for the Overview screen. Renders the four
 * authentication + transport surfaces in one glance so the operator
 * can answer "is Node-RED exposed safely?" without leaving Overview.
 *
 * Each row is a <StatusChip> backed by the slice B CVA variants. The
 * tile carries a single deep-link action (Go to Security) — embedded
 * forms land on the dedicated Configuration / Security screen.
 */

export interface SecurityPostureSurface {
  id: 'adminAuth' | 'httpNodeAuth' | 'httpStaticAuth' | 'requireHttps';
  enabled: boolean;
}

export interface SecurityPostureTileProps {
  /** Whether each security surface is currently configured. */
  adminAuth: boolean;
  httpNodeAuth: boolean;
  httpStaticAuth: boolean;
  requireHttps: boolean;
}

type SurfaceKey = 'adminAuth' | 'httpNodeAuth' | 'httpStaticAuth' | 'requireHttps';

interface SurfaceRow {
  key: SurfaceKey;
  labelKey: string;
  enabledKey: string;
  disabledKey: string;
}

const SURFACE_ROWS: ReadonlyArray<SurfaceRow> = [
  {
    key: 'adminAuth',
    labelKey: 'dashboard:overviewTiles.securityPosture.surfaces.adminAuth',
    enabledKey: 'dashboard:overviewTiles.securityPosture.states.enabled',
    disabledKey: 'dashboard:overviewTiles.securityPosture.states.disabled',
  },
  {
    key: 'httpNodeAuth',
    labelKey: 'dashboard:overviewTiles.securityPosture.surfaces.httpNodeAuth',
    enabledKey: 'dashboard:overviewTiles.securityPosture.states.enabled',
    disabledKey: 'dashboard:overviewTiles.securityPosture.states.disabled',
  },
  {
    key: 'httpStaticAuth',
    labelKey: 'dashboard:overviewTiles.securityPosture.surfaces.httpStaticAuth',
    enabledKey: 'dashboard:overviewTiles.securityPosture.states.enabled',
    disabledKey: 'dashboard:overviewTiles.securityPosture.states.disabled',
  },
  {
    key: 'requireHttps',
    labelKey: 'dashboard:overviewTiles.securityPosture.surfaces.requireHttps',
    enabledKey: 'dashboard:overviewTiles.securityPosture.states.enabled',
    disabledKey: 'dashboard:overviewTiles.securityPosture.states.disabled',
  },
];

export function SecurityPostureTile({
  adminAuth,
  httpNodeAuth,
  httpStaticAuth,
  requireHttps,
}: SecurityPostureTileProps) {
  const { t } = useT();
  const values: Record<SurfaceKey, boolean> = {
    adminAuth,
    httpNodeAuth,
    httpStaticAuth,
    requireHttps,
  };

  // Overall posture: success only when ALL surfaces are enabled. Two
  // surfaces off = warning. Three or more = danger.
  const enabledCount = Object.values(values).filter(Boolean).length;
  let overallVariant: 'success' | 'warning' | 'danger';
  if (enabledCount === 4) overallVariant = 'success';
  else if (enabledCount >= 2) overallVariant = 'warning';
  else overallVariant = 'danger';

  const overallLabelKey = `dashboard:overviewTiles.securityPosture.overall.${overallVariant}`;

  return (
    <article
      data-testid="overview-security-posture-tile"
      className="card surface-card border border-border p-6"
      aria-labelledby="overview-security-posture-title"
    >
      <header className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          {overallVariant === 'success' ? (
            <ShieldCheck className="h-6 w-6 text-success" aria-hidden="true" />
          ) : (
            <ShieldOff className="h-6 w-6 text-warning" aria-hidden="true" />
          )}
          <div className="min-w-0">
            <h2 id="overview-security-posture-title" className="text-base font-semibold text-base-content">
              {t('dashboard:overviewTiles.securityPosture.title')}
            </h2>
            <p className="mt-0.5 text-xs text-base-content/65">
              {t('dashboard:overviewTiles.securityPosture.subtitle')}
            </p>
          </div>
        </div>
        <StatusChip variant={overallVariant} size="sm" ariaLabel={t(overallLabelKey)}>
          {t(overallLabelKey)}
        </StatusChip>
      </header>

      <ul className="mt-4 space-y-2">
        {SURFACE_ROWS.map((row) => {
          const enabled = values[row.key];
          const variant = enabled ? 'success' : 'danger';
          return (
            <li
              key={row.key}
              data-testid={`overview-security-${row.key}`}
              className="flex items-center justify-between gap-3 rounded-xl border border-border/60 bg-base-200/30 px-3 py-2"
            >
              <span className="text-sm text-base-content">{t(row.labelKey)}</span>
              <StatusChip variant={variant} size="sm">
                {t(enabled ? row.enabledKey : row.disabledKey)}
              </StatusChip>
            </li>
          );
        })}
      </ul>

      <Link
        to="/security"
        className="mt-4 inline-flex items-center gap-1 text-sm font-medium text-accent transition-colors hover:text-accent-hover focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-1 focus-visible:ring-offset-base-100"
        data-testid="overview-security-posture-link"
      >
        {t('dashboard:overviewTiles.securityPosture.cta')}
        <ChevronRight className="h-4 w-4" aria-hidden="true" />
      </Link>
    </article>
  );
}
