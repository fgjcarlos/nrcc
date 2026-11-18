import { ArrowRight } from 'lucide-react';
import { useT } from '@/i18n';
import { useConfigurationData } from '@/features/configuration/hooks/useConfigurationData';
import { useUsersData } from '@/features/auth/hooks/useUsersData';
import { LoadingBoundary } from '@/shared/components/feedback';
import { StatusChip } from '@/shared/components/ui';
import type { NodeRedConfigResponse } from '@/features/configuration/lib/configTransformers';
import type { HttpBasicAuthSurface } from './HttpBasicAuthBoundaryCard';
import { AdminAuthBoundaryCard } from './AdminAuthBoundaryCard';
import { HttpBasicAuthBoundaryCard } from './HttpBasicAuthBoundaryCard';
import { NrccAccessBoundaryCard } from './NrccAccessBoundaryCard';
import { SecuritySurface } from './SecuritySurface';
import { DashboardAccess } from '@/features/configuration/components/DashboardAccess';

/**
 * SecurityView — issue #766 slice E
 *
 * Slice A lifted the authentication surfaces out of /configuration
 * into a single SecurityCenter component. Slice E splits that centre
 * into four independent boundary cards so the operator can change
 * one surface without touching the others:
 *
 *   1. NrccAccessBoundaryCard — NRCC operator access (read-only
 *      summary + deep-link to /settings/users).
 *   2. AdminAuthBoundaryCard — Node-RED adminAuth (users +
 *      sessionExpiryTime).
 *   3. HttpBasicAuthBoundaryCard (surface httpNodeAuth) — Node HTTP auth.
 *   4. HttpBasicAuthBoundaryCard (surface httpStaticAuth) — Static auth.
 *
 *   5. DashboardAccess — Dashboard HTTP/Socket.IO policy
 *      (unchanged from slice A).
 *
 * Composition (T10): each surface is a flat row (name, state chip,
 * what it protects) with its form behind an Edit disclosure and a
 * stable anchor (/security#admin-auth etc.). Each keeps its own save
 * action. The admin/node/static one combined Save Security
 * Center button is gone; the dependencies between them (legacy
 * alias migration) are now surfaced inside the affected boundary
 * card as a confirmation dialog.
 */
export function SecurityView() {
  const { t } = useT();
  const data = useConfigurationData();
  const { users } = useUsersData({ enabled: Boolean(data.hostStatus) });

  const editable = data.hostStatus?.configuration?.editable === true;
  const expectedRevision = data.settingsDoc?.revision?.fingerprint;
  const refetch = () => {
    void data.refetchConfig();
    void data.refetchSettings();
  };

  // SAFETY: the basic-auth surfaces carry { user, pass } at runtime, while
  // `NodeRedConfigResponse` also describes the multi-account `adminAuth`
  // shape. The cards declare their own `HttpBasicAuthSurface` type, so the
  // boundary is explicit instead of an `as any` cast.
  //
  // The keys are the ones the backend reports: `httpNodeAuth` and
  // `httpStaticAuth`. The legacy aliases (`nodeHttpAuth` / `staticAuth`)
  // are only ever written by hand in settings.js — the backend never
  // reports them — so reading them here made the cards show an
  // unconfigured surface for a host that actually had one.
  const config = data.config as NodeRedConfigResponse | null;
  const httpNodeConfig = (config?.httpNodeAuth ?? null) as HttpBasicAuthSurface;
  const httpStaticConfig = (config?.httpStaticAuth ?? null) as HttpBasicAuthSurface;

  const securityViewState = data.configLoading || data.settingsLoading
    ? { status: 'pending' as const }
    : data.configError
      ? {
          status: 'error' as const,
          error: data.configError,
          onRetry: refetch,
        }
      : { status: 'success' as const };

  return (
    <LoadingBoundary state={securityViewState}>
    <section
      data-testid="security-view"
      aria-labelledby="security-view-title"
    >
      <header className="space-y-2 border-b border-ds-border-subtle pb-3">
        <h1 id="security-view-title" className="text-xl font-bold text-ds-text-primary">
          {t('security:title')}
        </h1>
        <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('security:subtitle')}</p>
        <figure className="space-y-1">
          <ol className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-ds-text-secondary">
            <li className="font-semibold text-ds-text-primary">{t('configuration:topology.nodeNrcc')}</li>
            <li className="inline-flex items-center gap-1">
              <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
              {t('configuration:topology.arrowNrccSettings')}
            </li>
            <li className="font-mono font-semibold text-ds-text-primary">{t('configuration:topology.nodeSettings')}</li>
            <li className="inline-flex items-center gap-1">
              <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
              {t('configuration:topology.arrowSettingsNodeRed')}
            </li>
            <li className="font-semibold text-ds-text-primary">{t('configuration:topology.nodeRed')}</li>
          </ol>
          <figcaption className="text-xs text-ds-text-secondary">{t('configuration:topology.subtitle')}</figcaption>
        </figure>
        <p className="max-w-[75ch] text-xs text-ds-text-secondary" data-testid="security-password-note">
          {t('security:passwordNote')}
        </p>
      </header>

      <NrccAccessBoundaryCard users={users} />

      <AdminAuthBoundaryCard
        config={config?.adminAuth ?? null}
        rawSettingsContent={data.rawSettingsContent ?? ''}
        expectedRevision={expectedRevision}
        editable={editable}
        onApplied={refetch}
      />

      <HttpBasicAuthBoundaryCard
        surface="httpNodeAuth"
        config={httpNodeConfig}
        rawSettingsContent={data.rawSettingsContent ?? ''}
        expectedRevision={expectedRevision}
        editable={editable}
        onApplied={refetch}
      />

      <HttpBasicAuthBoundaryCard
        surface="httpStaticAuth"
        config={httpStaticConfig}
        rawSettingsContent={data.rawSettingsContent ?? ''}
        expectedRevision={expectedRevision}
        editable={editable}
        onApplied={refetch}
      />

      <SecuritySurface
        anchor="dashboard-access"
        testId="boundary-dashboard-access"
        title={t('security:dashboardAccess.title')}
        summary={t('security:dashboardAccess.summary')}
        status={
          // /config does not report the dashboard policy, so its state is unknown here.
          <StatusChip variant="neutral" size="sm">
            {t('security:dashboardAccess.notReported')}
          </StatusChip>
        }
      >
        <DashboardAccess
          editable={editable}
          expectedRevision={expectedRevision}
          onApplied={refetch}
        />
      </SecuritySurface>
    </section>
    </LoadingBoundary>
  );
}
