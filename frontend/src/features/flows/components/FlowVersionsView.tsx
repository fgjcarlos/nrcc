import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { flowService } from '../services/flowService';
import type { FlowVersionEntry, FlowDiff } from '../types';

export function FlowVersionsView() {
  const queryClient = useQueryClient();
  const [selectedVersions, setSelectedVersions] = useState<[string, string] | null>(null);
  const [revertTarget, setRevertTarget] = useState<string | null>(null);

  const { data: versions = [], isLoading } = useQuery({
    queryKey: ['flow-versions'],
    queryFn: flowService.getVersions,
    refetchInterval: 30_000,
  });

  const { data: diff, isLoading: diffLoading } = useQuery({
    queryKey: ['flow-diff', selectedVersions],
    queryFn: () =>
      selectedVersions ? flowService.getVersionDiff(selectedVersions[0], selectedVersions[1]) : null,
    enabled: !!selectedVersions,
  });

  const revertMutation = useMutation({
    mutationFn: flowService.revertToVersion,
    onSuccess: () => {
      toast.success('Flows reverted successfully');
      queryClient.invalidateQueries({ queryKey: ['flow-versions'] });
      setRevertTarget(null);
    },
    onError: () => toast.error('Failed to revert flows'),
  });

  const snapshotMutation = useMutation({
    mutationFn: flowService.captureSnapshot,
    onSuccess: () => {
      toast.success('Snapshot captured');
      queryClient.invalidateQueries({ queryKey: ['flow-versions'] });
    },
    onError: () => toast.error('Failed to capture snapshot'),
  });

  const handleCompare = (fromIdx: number) => {
    if (fromIdx + 1 < versions.length) {
      setSelectedVersions([versions[fromIdx + 1].id, versions[fromIdx].id]);
    }
  };

  if (isLoading) {
    return <div className="p-6 text-muted-foreground">Loading versions...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-base-content">Flow Versions</h2>
        <button
          onClick={() => snapshotMutation.mutate()}
          disabled={snapshotMutation.isPending}
          className="action-btn-secondary text-sm"
        >
          {snapshotMutation.isPending ? 'Capturing...' : 'Capture Snapshot'}
        </button>
      </div>

      {versions.length === 0 ? (
        <p className="text-muted-foreground">No versions captured yet. Flow changes are detected automatically.</p>
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
        />
      )}

      {selectedVersions && (
        <DiffPanel diff={diff ?? null} loading={diffLoading} onClose={() => setSelectedVersions(null)} />
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
}: {
  version: FlowVersionEntry;
  isLatest: boolean;
  onCompare: () => void;
  onRevert: () => void;
  canCompare: boolean;
}) {
  const date = version.timestamp ? new Date(version.timestamp).toLocaleString() : version.id;

  return (
    <div className="surface-panel flex items-center justify-between border border-border p-3 rounded-xl">
      <div className="flex items-center gap-4">
        <div>
          <span className="text-sm font-medium text-base-content">{date}</span>
          {isLatest && (
            <span className="ml-2 rounded bg-primary/20 px-2 py-0.5 text-xs text-primary">latest</span>
          )}
        </div>
        <span className="text-xs text-muted-foreground">{version.nodeCount} nodes</span>
        <span className="text-xs text-muted-foreground">{(version.size / 1024).toFixed(1)} KB</span>
        <span className="font-mono text-xs text-muted-foreground">{version.hash}</span>
      </div>
      <div className="flex gap-2">
        {canCompare && (
          <button onClick={onCompare} className="action-btn-secondary text-xs">
            Diff
          </button>
        )}
        {!isLatest && (
          <button onClick={onRevert} className="action-btn-secondary text-xs text-warning">
            Revert
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
}: {
  diff: FlowDiff | null;
  loading: boolean;
  onClose: () => void;
}) {
  const isEmpty = diff && !diff.added?.length && !diff.removed?.length && !diff.modified?.length;

  return (
    <div className="surface-panel border border-border rounded-xl p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-bold text-base-content">Changes</h3>
        <button onClick={onClose} className="text-muted-foreground hover:text-base-content text-sm">
          Close
        </button>
      </div>

      {loading && <p className="text-muted-foreground text-sm">Computing diff...</p>}
      {isEmpty && <p className="text-muted-foreground text-sm">No differences found.</p>}

      {diff?.added && diff.added.length > 0 && (
        <div>
          <h4 className="text-sm font-medium text-success mb-1">+ Added ({diff.added.length})</h4>
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
          <h4 className="text-sm font-medium text-error mb-1">- Removed ({diff.removed.length})</h4>
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
          <h4 className="text-sm font-medium text-warning mb-1">~ Modified ({diff.modified.length})</h4>
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
}: {
  versionId: string;
  isPending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <div className="modal-overlay" onClick={(e) => e.target === e.currentTarget && onCancel()}>
      <div className="surface-panel w-full max-w-md border border-border p-6 shadow-glow">
        <h3 className="text-lg font-bold text-base-content mb-2">Confirm Revert</h3>
        <p className="text-sm text-muted-foreground mb-4">
          This will replace the current flows.json with the selected version. A snapshot of the current
          state will be captured first.
        </p>
        <p className="text-xs font-mono text-muted-foreground mb-4">{versionId}</p>
        <div className="flex justify-end gap-2">
          <button onClick={onCancel} disabled={isPending} className="action-btn-secondary">
            Cancel
          </button>
          <button onClick={onConfirm} disabled={isPending} className="action-btn-primary">
            {isPending ? 'Reverting...' : 'Revert'}
          </button>
        </div>
      </div>
    </div>
  );
}
