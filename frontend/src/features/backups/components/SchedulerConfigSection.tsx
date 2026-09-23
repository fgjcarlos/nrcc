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
    <div className="surface-card p-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-base-content">{t('backups:scheduler.schedulerTitle')}</h2>
        <p className="text-sm text-base-content/65">
          {t('backups:scheduler.schedulerDesc')}
        </p>
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
    </div>
  );
}
