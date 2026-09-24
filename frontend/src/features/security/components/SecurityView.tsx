import { Shield } from 'lucide-react';
import { useT } from '@/i18n';
import { SecurityCenter } from '@/features/configuration/components/SecurityCenter';
import { DashboardAccess } from '@/features/configuration/components/DashboardAccess';
import { useConfigurationData } from '@/features/configuration/hooks/useConfigurationData';

/**
 * Dedicated Security section (slice A of issue #766).
 *
 * Lifts the authentication surfaces (Node-RED adminAuth, httpNodeAuth,
 * static auth, dashboard access) out of /configuration's Authentication tab
 * so the Sidebar can promote them to a first-class section.
 *
 * Functional contracts come from #760 + #761; this view is purely a
 * layout/routing change. Component props match what ConfigurationView
 * passed when it owned this block.
 */
export function SecurityView() {
  const { t } = useT();
  const data = useConfigurationData();

  const editable = data.hostStatus?.configuration?.editable === true;
  const expectedRevision = data.settingsDoc?.revision?.fingerprint;
  const refetch = () => {
    void data.refetchConfig();
    void data.refetchSettings();
  };

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

      <SecurityCenter
        config={data.config}
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
