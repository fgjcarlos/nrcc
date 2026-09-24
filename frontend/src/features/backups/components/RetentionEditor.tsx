import { useState, useEffect } from 'react';
import { LoaderCircle } from 'lucide-react';
import { useT } from '@/i18n';

export interface RetentionEditorProps {
  retentionManual: number;
  retentionAuto: number;
  retentionPreRestore: number;
  onSave: (manual: number, auto: number, preRestore: number) => void;
  isSaving?: boolean;
}

export function RetentionEditor({
  retentionManual,
  retentionAuto,
  retentionPreRestore,
  onSave,
  isSaving = false,
}: RetentionEditorProps) {
  const { t } = useT();
  const [manual, setManual] = useState(retentionManual);
  const [auto, setAuto] = useState(retentionAuto);
  const [preRestore, setPreRestore] = useState(retentionPreRestore);

  useEffect(() => {
    setManual(retentionManual);
    setAuto(retentionAuto);
    setPreRestore(retentionPreRestore);
  }, [retentionManual, retentionAuto, retentionPreRestore]);

  const handleSave = () => {
    onSave(manual, auto, preRestore);
  };

  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-3">
        <label className="space-y-2">
          <span className="text-sm font-medium text-base-content">{t('backups:manualBackups')}</span>
          <input
            type="number"
            min={1}
            max={3650}
            value={manual}
            onChange={(e) => setManual(Number(e.target.value) || 1)}
            className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
          <p className="text-xs text-base-content/55">{t('backups:daysKeepManual')}</p>
        </label>

        <label className="space-y-2">
          <span className="text-sm font-medium text-base-content">{t('backups:automaticBackups')}</span>
          <input
            type="number"
            min={1}
            max={3650}
            value={auto}
            onChange={(e) => setAuto(Number(e.target.value) || 1)}
            className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
          <p className="text-xs text-base-content/55">{t('backups:daysKeepAutomatic')}</p>
        </label>

        <label className="space-y-2">
          <span className="text-sm font-medium text-base-content">{t('backups:preRestoreSnapshots')}</span>
          <input
            type="number"
            min={1}
            max={3650}
            value={preRestore}
            onChange={(e) => setPreRestore(Number(e.target.value) || 1)}
            className="glass-panel w-full rounded-xl border border-border px-3 py-2 text-base-content focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
          <p className="text-xs text-base-content/55">{t('backups:daysKeepPreRestore')}</p>
        </label>
      </div>

      <div className="flex justify-end gap-3">
        {isSaving && (
          <div className="inline-flex items-center gap-2 rounded-xl border border-primary/20 bg-primary/8 px-3 py-2 text-xs text-base-content/75">
            <LoaderCircle className="h-3.5 w-3.5 animate-spin text-primary" />
            {t('common:saving')}...
          </div>
        )}
        <button
          onClick={handleSave}
          disabled={isSaving}
          className="action-btn-primary"
        >
          {isSaving && <LoaderCircle className="h-4 w-4 animate-spin" />}
          {t('backups:saveRetentionPolicy')}
        </button>
      </div>
    </div>
  );
}
