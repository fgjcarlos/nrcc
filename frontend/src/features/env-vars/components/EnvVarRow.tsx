import { type EnvVar } from '@/features/env-vars/services/envService';
import { Eye, EyeOff, Trash2, Pencil } from 'lucide-react';

export function EnvVarRow({ envVar, onDelete, onToggleSecret, onEdit, showSecret }: {
  envVar: EnvVar;
  onDelete: (key: string) => void;
  onToggleSecret: (key: string) => void;
  onEdit: (envVar: EnvVar) => void;
  showSecret: boolean;
}) {
  return (
    <tr className="table-row-hover">
      <td className="px-4 py-3 font-mono text-sm text-base-content">{envVar.key}</td>
      <td className="px-4 py-3 font-mono text-sm text-base-content">
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
      <td className="px-4 py-3">
        <span className={`rounded-full px-2 py-1 text-xs ${
          envVar.type === 'secret' ? 'bg-error/15 text-error-content' :
          envVar.type === 'boolean' ? 'bg-info/15 text-info-content' :
          envVar.type === 'number' ? 'bg-success/15 text-success-content' :
          'bg-base-300/70 text-base-content'
        }`}>{envVar.type}</span>
      </td>
      <td className="px-4 py-3 text-sm text-base-content/60">{envVar.description || '-'}</td>
      <td className="px-4 py-3 text-right">
        <div className="flex items-center justify-end gap-2">
          <button onClick={() => onEdit(envVar)} className="rounded p-2 text-primary transition-colors hover:bg-primary/10">
            <Pencil className="w-4 h-4" />
          </button>
          <button onClick={() => onDelete(envVar.key)} className="rounded p-2 text-error transition-colors hover:bg-error/10">
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </td>
    </tr>
  );
}

export default EnvVarRow;
