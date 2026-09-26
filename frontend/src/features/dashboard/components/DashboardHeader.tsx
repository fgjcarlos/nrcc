import { useT } from '@/i18n';

interface DashboardHeaderProps {
  edgeMode?: boolean;
}

/**
 * DashboardHeader — issue #766 slice C
 *
 * Page-level header for the Overview screen. Shows the screen title
 * and a "Live telemetry" tag for operator context.
 *
 * Edge-mode used to be rendered here; it now lives in the persistent
 * Header (slice C) so the operator sees the deployment state on
 * every authenticated page, not only on Overview.
 */
export function DashboardHeader(_props: DashboardHeaderProps) {
  const { t } = useT();
  return (
    <div className="flex items-end justify-between gap-4">
      <div>
        <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">{t('dashboard:systemOverview')}</p>
        <h1 className="text-3xl font-bold tracking-tight text-base-content">{t('dashboard:overview')}</h1>
      </div>
      <div className="flex items-center gap-2">
        <div className="hidden rounded-full bg-base-300/60 px-4 py-2 text-xs font-medium text-base-content/70 md:block">
          {t('dashboard:liveTelemetry')}
        </div>
      </div>
    </div>
  );
}
