import { Shield } from 'lucide-react';
import { useT } from '@/i18n';
import { useConfigurationData } from '@/features/configuration/hooks/useConfigurationData';
import { useUsersData } from '@/features/auth/hooks/useUsersData';
import type { NodeRedConfigResponse } from '@/features/configuration/lib/configTransformers';
import type { HttpBasicAuthSurface } from './HttpBasicAuthBoundaryCard';
import { AdminAuthBoundaryCard } from './AdminAuthBoundaryCard';
import { HttpBasicAuthBoundaryCard } from './HttpBasicAuthBoundaryCard';
import { NrccAccessBoundaryCard } from './NrccAccessBoundaryCard';
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
 * Composition: 5 cards in order, each independent, each with its
 * own save action. The admin/node/static one combined Save Security
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

  // SAFETY: NodeRedConfigResponse types nodeHttpAuth / staticAuth as
  // CredentialsAuthResponse (a multi-user credentials shape), but the
  // basic-auth surfaces carry { user, pass } at runtime. SecurityCenter
  // worked around this with a per-cast `as Partial<BasicAuth>` for
  // years; slice E formalises the boundary by declaring an
  // HttpBasicAuthSurface on the card. The runtime payload matches the
  // card's expectations; the type-system gap is a known schema drift.
  const config = data.config as NodeRedConfigResponse | null;
  const httpNodeConfig = (config?.nodeHttpAuth ?? null) as HttpBasicAuthSurface;
  const httpStaticConfig = (config?.staticAuth ?? null) as HttpBasicAuthSurface;

  return (
    <section
      data-testid="security-view"
      className="space-y-6"
      aria-labelledby="security-view-title"
    >
      <header className="flex items-center gap-3 border-b border-border pb-4">
        <Shield className="h-6 w-6 stroke-[1.6] text-base-content/70" aria-hidden="true" />
        <div className="min-w-0">
          <h1 id="security-view-title" className="text-xl font-semibold text-base-content">
            {t('security:title')}
          </h1>
          <p className="text-sm text-base-content/65">{t('security:subtitle')}</p>
        </div>
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

      <div className="border-t border-border" />

      <DashboardAccess
        editable={editable}
        expectedRevision={expectedRevision}
        onApplied={refetch}
      />
    </section>
  );
}
