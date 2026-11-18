import { useState, useEffect } from 'react';
import { LoaderCircle } from 'lucide-react';
import { useT } from '@/i18n';
import { Button, Input } from '@/shared/components/ui';

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

  const fields = [
    { key: 'manual', label: t('backups:retention.manual'), value: manual, set: setManual },
    { key: 'auto', label: t('backups:retention.auto'), value: auto, set: setAuto },
    { key: 'preRestore', label: t('backups:retention.preRestore'), value: preRestore, set: setPreRestore },
  ];

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-3">
        {fields.map((field) => (
          <Input
            key={field.key}
            id={`backup-retention-${field.key}`}
            label={field.label}
            type="number"
            inputMode="numeric"
            min={1}
            max={3650}
            value={field.value}
            onChange={(e) => field.set(Number(e.target.value) || 1)}
            className="tabular-nums"
          />
        ))}
      </div>

      <div className="flex justify-end">
        <Button type="button" onClick={handleSave} disabled={isSaving} aria-busy={isSaving}>
          {isSaving && <LoaderCircle className="h-4 w-4 animate-spin motion-reduce:animate-none" aria-hidden="true" />}
          {t('backups:retention.save')}
        </Button>
      </div>
    </div>
  );
}
