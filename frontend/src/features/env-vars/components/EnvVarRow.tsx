import { type EnvVar } from '@/features/env-vars/services/envService';
import { Eye, EyeOff, Trash2, Pencil } from 'lucide-react';
import { useT } from '@/i18n';

export function EnvVarRow({ envVar, onDelete, onToggleSecret, onEdit, showSecret }: {
  envVar: EnvVar;
  onDelete: (key: string) => void;
  onToggleSecret: (key: string) => void;
  onEdit: (envVar: EnvVar) => void;
  showSecret: boolean;
}) {
  const { t } = useT();
  return (
    <tr>
      <td data-label={t('env-vars:colKey')} className="font-mono">{envVar.key}</td>
      <td data-label={t('env-vars:colValue')} className="font-mono [overflow-wrap:anywhere]">
        <div className="flex items-center gap-2">
          {envVar.type === 'secret' ? (
            <>
              <span>{showSecret ? envVar.value : '••••••••'}</span>
              <button onClick={() => onToggleSecret(envVar.key)} className="text-base-content/60 hover:text-base-content">
                {showSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </>
          ) : (
            envVar.value
          )}
        </div>
      </td>
      <td data-label={t('env-vars:colType')}>
        <span className={`whitespace-nowrap rounded-full px-2 py-1 text-xs ${
          envVar.type === 'secret' ? 'bg-error/15 text-ds-on-tint-error' :
          envVar.type === 'boolean' ? 'bg-info/15 text-ds-on-tint-info' :
          envVar.type === 'number' ? 'bg-success/15 text-ds-on-tint-success' :
          'bg-base-300/70 text-base-content'
        }`}>{envVar.type}</span>
        {envVar.type === 'secret' && (
          <span className="ml-2 text-xs text-base-content/50">{t('env-vars:runtimeOnly')}</span>
        )}
      </td>
      <td data-label={t('env-vars:colSource')}>
        <span className="whitespace-nowrap rounded-full bg-base-300/70 px-2 py-1 text-xs text-base-content">
          {envVar.source === 'node-red' ? 'Node-RED' : 'NRCC'}
        </span>
      </td>
      <td data-label={t('env-vars:colDescription')} className="text-ds-text-secondary">{envVar.description || '-'}</td>
      <td data-label={t('env-vars:colActions')} className="text-right">
        <div className="flex items-center justify-end gap-2">
          <button
            onClick={() => onEdit(envVar)}
            aria-label={`${t('common:edit')} ${envVar.key}`}
            className="rounded p-2 text-primary transition-colors hover:bg-primary/10"
          >
            <Pencil className="w-4 h-4" aria-hidden="true" />
          </button>
          <button
            onClick={() => onDelete(envVar.key)}
            aria-label={`${t('common:delete')} ${envVar.key}`}
            className="rounded p-2 text-error transition-colors hover:bg-error/10"
          >
            <Trash2 className="w-4 h-4" aria-hidden="true" />
          </button>
        </div>
      </td>
    </tr>
  );
}

export default EnvVarRow;
