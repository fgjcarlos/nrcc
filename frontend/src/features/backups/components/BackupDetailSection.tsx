import { LoaderCircle, X } from 'lucide-react';
import type { BackupSummary } from '@/features/backups/services';
import { formatBackupSize, getBackupSummary } from '@/features/backups/lib/formatters';

interface BackupDetail {
  files: Array<{
    path: string;
    size: number;
    checksum: string;
  }>;
}

interface BackupDetailSectionProps {
  backup: BackupSummary;
  detail: BackupDetail | undefined;
  isLoading: boolean;
  onClose: () => void;
}

const typeLabels: Record<BackupSummary['type'], string> = {
  manual: 'Manual',
  auto: 'Auto',
  'pre-restore': 'Pre-restore',
};

export function BackupDetailSection(props: BackupDetailSectionProps) {
  const { backup, detail, isLoading, onClose } = props;

  if (!detail && !isLoading) {
    return null;
  }

  return (
    <div className="surface-card p-6">
      {/* Header */}
      <div className="mb-4 flex items-center justify-between gap-4">
        <div>
          <h3 className="text-lg font-semibold text-base-content">Detalle del backup</h3>
          <p className="text-sm text-base-content/65">{getBackupSummary(backup, typeLabels)}</p>
        </div>
        <button onClick={onClose} className="icon-button text-base-content/60 hover:text-base-content">
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* Loading State */}
      {isLoading && !detail ? (
        <div className="flex items-center justify-center py-8">
          <div className="text-center">
            <LoaderCircle className="mx-auto mb-3 h-10 w-10 animate-spin text-base-content/40" />
            <p className="text-sm text-base-content/65">Cargando detalle...</p>
          </div>
        </div>
      ) : detail && detail.files && detail.files.length > 0 ? (
        /* File List */
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs text-base-content/60">
                <th className="py-3 pr-4">Ruta</th>
                <th className="py-3 pr-4 text-right">Tamaño</th>
                <th className="py-3 pr-4">Checksum</th>
              </tr>
            </thead>
            <tbody>
              {detail.files.map((file, idx) => (
                <tr key={idx} className="border-b border-border/50 hover:bg-base-200/15">
                  <td className="py-2 pr-4 font-mono text-base-content">{file.path}</td>
                  <td className="py-2 pr-4 text-right text-base-content/70">{formatBackupSize(file.size)}</td>
                  <td className="py-2 pr-4 font-mono text-xs text-base-content/50">{file.checksum.slice(0, 16)}…</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        /* No Files */
        <div className="rounded-lg border border-border bg-base-200/20 p-4 text-center">
          <p className="text-sm text-base-content/65">No hay archivos en este backup</p>
        </div>
      )}
    </div>
  );
}
