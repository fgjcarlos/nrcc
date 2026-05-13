import type { BackupConfig } from '@/features/backups/services';
import { RetentionEditor } from './RetentionEditor';

interface RetentionPolicySectionProps {
  retentionManual: BackupConfig['retentionManual'];
  retentionAuto: BackupConfig['retentionAuto'];
  retentionPreRestore: BackupConfig['retentionPreRestore'];
  onSave: (manual: number, auto: number, preRestore: number) => void;
  isSaving?: boolean;
}

export function RetentionPolicySection(props: RetentionPolicySectionProps) {
  const { retentionManual, retentionAuto, retentionPreRestore, isSaving, onSave } = props;

  return (
    <div className="surface-card p-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-base-content">Política de retención</h2>
        <p className="text-sm text-base-content/65">Cuántos días guardar cada tipo de backup antes de eliminar automáticamente</p>
      </div>

      <RetentionEditor
        retentionManual={retentionManual}
        retentionAuto={retentionAuto}
        retentionPreRestore={retentionPreRestore}
        isSaving={isSaving}
        onSave={onSave}
      />
    </div>
  );
}
