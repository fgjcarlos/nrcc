import type { BackupConfig } from '@/features/backups/services';
import { useT } from '@/i18n';
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
  const { t } = useT();

  return (
    <section aria-labelledby="backups-retention-title" className="space-y-3 py-4">
      <div className="space-y-1">
        <h2 id="backups-retention-title" className="text-sm font-semibold text-ds-text-primary">
          {t('backups:retention.title')}
        </h2>
        <p className="text-xs text-ds-text-secondary">{t('backups:retention.desc')}</p>
      </div>

      <RetentionEditor
        retentionManual={retentionManual}
        retentionAuto={retentionAuto}
        retentionPreRestore={retentionPreRestore}
        isSaving={isSaving}
        onSave={onSave}
      />
    </section>
  );
}
