import { EdgeModeBadge } from './EdgeModeBadge';
import { useT } from '@/i18n';

interface DashboardHeaderProps {
  edgeMode?: boolean;
}

export function DashboardHeader({ edgeMode }: DashboardHeaderProps) {
  const { t } = useT();
  return (
    <div className="flex items-end justify-between gap-4">
      <div>
        <p className="text-xs uppercase tracking-[0.28em] text-base-content/50">{t('dashboard:systemOverview')}</p>
        <h1 className="text-3xl font-bold tracking-tight text-base-content">{t('dashboard:overview')}</h1>
      </div>
      <div className="flex items-center gap-2">
        <EdgeModeBadge enabled={edgeMode} />
        <div className="hidden rounded-full bg-base-300/60 px-4 py-2 text-xs font-medium text-base-content/70 md:block">
          {t('dashboard:liveTelemetry')}
        </div>
      </div>
    </div>
  );
}
