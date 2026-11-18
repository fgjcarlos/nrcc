import type { BackupConfig } from '@/features/backups/services';
import { CronBuilder, type SaveState, type PresetType } from './CronBuilder';
import { useT } from '@/i18n';

interface SchedulerConfigSectionProps {
  configDraft: BackupConfig;
  saveState: SaveState;
  saveError: string | undefined;
  onChange: (cron: string) => void;
  onPresetChange: (preset: PresetType) => void;
  onSave: () => void;
}

export function SchedulerConfigSection(props: SchedulerConfigSectionProps) {
  const { t } = useT();
  const { configDraft, saveState, saveError, onChange, onPresetChange, onSave } = props;
  const schedule = (configDraft.schedule ?? 'disabled') as PresetType;

  return (
    <section aria-labelledby="backups-schedule-title" className="space-y-3 py-4">
      <div className="space-y-1">
        <h2 id="backups-schedule-title" className="text-sm font-semibold text-ds-text-primary">
          {t('backups:scheduler.title')}
        </h2>
        <p className="text-xs text-ds-text-secondary">{t('backups:scheduler.desc')}</p>
      </div>

      <CronBuilder
        schedule={schedule}
        customSchedule={configDraft.customSchedule}
        onChange={onChange}
        onPresetChange={onPresetChange}
        onSave={onSave}
        saveState={saveState}
        saveError={saveError}
      />
    </section>
  );
}
