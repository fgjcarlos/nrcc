import type { BackupConfig } from '@/features/backups/services';
import { CronBuilder, type SaveState, type PresetType } from './CronBuilder';

interface SchedulerConfigSectionProps {
  configDraft: BackupConfig;
  saveState: SaveState;
  saveError: string | undefined;
  onChange: (cron: string) => void;
  onPresetChange: (preset: PresetType) => void;
  onSave: () => void;
}

export function SchedulerConfigSection(props: SchedulerConfigSectionProps) {
  const { configDraft, saveState, saveError, onChange, onPresetChange, onSave } = props;
  const schedule = (configDraft.schedule ?? 'disabled') as PresetType;

  return (
    <div className="surface-card p-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-base-content">Configuración de horario</h2>
        <p className="text-sm text-base-content/65">
          Elige una frecuencia predeterminada o programa un backup único en una fecha y hora concretas.
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
