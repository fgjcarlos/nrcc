import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { flowService } from '../services/flowService';
import type { FlowVersionEntry, FlowDiff } from '../types';

import { queryKeys } from '@/shared/lib/queryKeys';
import { useT } from '@/i18n';
export function FlowVersionsView() {
  const queryClient = useQueryClient();
  const { t } = useT();
  const [selectedVersions, setSelectedVersions] = useState<[string, string] | null>(null);
  const [revertTarget, setRevertTarget] = useState<string | null>(null);

  const { data: versions = [], isLoading } = useQuery({
    queryKey: queryKeys.flows.versions,
    queryFn: flowService.getVersions,
    refetchInterval: 30_000,
  });

  const { data: diff, isLoading: diffLoading } = useQuery({
    queryKey: queryKeys.flows.diff(selectedVersions),
    queryFn: () =>
      selectedVersions ? flowService.getVersionDiff(selectedVersions[0], selectedVersions[1]) : null,
    enabled: !!selectedVersions,
  });

  const revertMutation = useMutation({
    mutationFn: flowService.revertToVersion,
    onSuccess: () => {
      toast.success(t('flows:revertSucceeded'));
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.versions });
      setRevertTarget(null);
    },
    onError: () => toast.error(t('flows:revertFailed')),
  });

  const snapshotMutation = useMutation({
    mutationFn: flowService.captureSnapshot,
    onSuccess: () => {
      toast.success(t('flows:snapshotCaptured'));
      queryClient.invalidateQueries({ queryKey: queryKeys.flows.versions });
    },
    onError: () => toast.error(t('flows:snapshotFailed')),
  });

  const handleCompare = (fromIdx: number) => {
    if (fromIdx + 1 < versions.length) {
      setSelectedVersions([versions[fromIdx + 1].id, versions[fromIdx].id]);
    }
  };

  if (isLoading) {
    return <div className="p-6 text-muted-foreground">{t('flows:loadingVersions')}</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-base-content">{t('flows:versionsTitle')}</h2>
        <button
          onClick={() => snapshotMutation.mutate()}
          disabled={snapshotMutation.isPending}
          className="action-btn-secondary text-sm"
        >
          {snapshotMutation.isPending ? t('flows:capturing') : t('flows:captureSnapshot')}
        </button>
      </div>

      {versions.length === 0 ? (
        <p className="text-muted-foreground">{t('flows:noVersions')}</p>
      ) : (
        <div className="space-y-2">
          {versions.map((v, idx) => (
            <VersionRow
              key={v.id}
              version={v}
              isLatest={idx === 0}
              onCompare={() => handleCompare(idx)}
              onRevert={() => setRevertTarget(v.id)}
              canCompare={idx + 1 < versions.length}
              t={t}
            />
          ))}
        </div>
      )}

      {revertTarget && (
        <RevertConfirm
          versionId={revertTarget}
          isPending={revertMutation.isPending}
          onConfirm={() => revertMutation.mutate(revertTarget)}
          onCancel={() => setRevertTarget(null)}
          t={t}
        />
      )}

      {selectedVersions && (
        <DiffPanel diff={diff ?? null} loading={diffLoading} onClose={() => setSelectedVersions(null)} t={t} />
      )}
    </div>
  );
}

function VersionRow({
  version,
  isLatest,
  onCompare,
  onRevert,
  canCompare,
  t,
}: {
  version: FlowVersionEntry;
  isLatest: boolean;
  onCompare: () => void;
  onRevert: () => void;
  canCompare: boolean;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const date = version.timestamp ? new Date(version.timestamp).toLocaleString() : version.id;

  return (
    <div className="surface-panel flex items-center justify-between border border-border p-3 rounded-xl">
      <div className="flex items-center gap-4">
        <div>
          <span className="text-sm font-medium text-base-content">{date}</span>
          {isLatest && (
            <span className="ml-2 rounded bg-primary/20 px-2 py-0.5 text-xs text-primary">{t('flows:latest')}</span>
          )}
        </div>
        <span className="text-xs text-muted-foreground">{t('flows:nodeCount', { count: version.nodeCount })}</span>
        <span className="text-xs text-muted-foreground">{(version.size / 1024).toFixed(1)} KB</span>
        <span className="font-mono text-xs text-muted-foreground">{version.hash}</span>
      </div>
      <div className="flex gap-2">
        {canCompare && (
          <button onClick={onCompare} className="action-btn-secondary text-xs">
            {t('flows:diff')}
          </button>
        )}
        {!isLatest && (
          <button onClick={onRevert} className="action-btn-secondary text-xs text-warning">
            {t('flows:revert')}
          </button>
        )}
      </div>
    </div>
  );
}

function DiffPanel({
  diff,
  loading,
  onClose,
  t,
}: {
  diff: FlowDiff | null;
  loading: boolean;
  onClose: () => void;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  const isEmpty = diff && !diff.added?.length && !diff.removed?.length && !diff.modified?.length;

  return (
    <div className="surface-panel border border-border rounded-xl p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-bold text-base-content">{t('flows:changes')}</h3>
        <button onClick={onClose} className="text-muted-foreground hover:text-base-content text-sm">
          {t('common:close')}
        </button>
      </div>

      {loading && <p className="text-muted-foreground text-sm">{t('flows:computingDiff')}</p>}
      {isEmpty && <p className="text-muted-foreground text-sm">{t('flows:noDifferences')}</p>}

      {diff?.added && diff.added.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-success mb-1">+ {t('flows:added', { count: diff.added.length })}</h4>
          {diff.added.map((n) => (
            <div key={n.id} className="text-sm text-muted-foreground ml-4">
              <span className="font-mono">{n.type}</span>
              {n.label && <span className="ml-2">{n.label}</span>}
            </div>
          ))}
        </div>
      )}

      {diff?.removed && diff.removed.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-error mb-1">- {t('flows:removed', { count: diff.removed.length })}</h4>
          {diff.removed.map((n) => (
            <div key={n.id} className="text-sm text-muted-foreground ml-4">
              <span className="font-mono">{n.type}</span>
              {n.label && <span className="ml-2">{n.label}</span>}
            </div>
          ))}
        </div>
      )}

      {diff?.modified && diff.modified.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-warning mb-1">~ {t('flows:modified', { count: diff.modified.length })}</h4>
          {diff.modified.map((n) => (
            <div key={n.id} className="text-sm text-muted-foreground ml-4">
              <span className="font-mono">{n.type}</span>
              {n.label && <span className="ml-2">{n.label}</span>}
              <span className="ml-2 text-xs">({n.changed.join(', ')})</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function RevertConfirm({
  versionId,
  isPending,
  onConfirm,
  onCancel,
  t,
}: {
  versionId: string;
  isPending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  t: (key: string, options?: Record<string, unknown>) => string;
}) {
  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onCancel()}>
      <div className="surface-panel w-full max-w-md border border-border p-6 shadow-glow">
        <h3 className="text-lg font-bold text-base-content mb-2">{t('flows:confirmRevert')}</h3>
        <p className="text-sm text-muted-foreground mb-4">{t('flows:revertDescription')}</p>
        <p className="text-xs font-mono text-muted-foreground mb-4">{versionId}</p>
        <div className="flex justify-end gap-2">
          <button onClick={onCancel} disabled={isPending} className="action-btn-secondary">
            {t('common:cancel')}
          </button>
          <button onClick={onConfirm} disabled={isPending} className="action-btn-primary">
            {isPending ? t('flows:reverting') : t('flows:revert')}
          </button>
        </div>
      </div>
    </div>
  );
}
