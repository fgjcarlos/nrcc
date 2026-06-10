import { useState } from 'react';
import { formatBytes, cn } from '@/shared/lib';
import { RotateCcw, Square } from 'lucide-react';
import { ConfirmationDialog, StateContainer } from '@/shared/components';
import { UI_COPY } from '@/shared/constants';
import { useDockerData, useDockerActions } from '@/features/docker/hooks';

export function DockerView() {
  // Confirmation dialog state
  const [confirmConfig, setConfirmConfig] = useState<{
    isOpen: boolean;
    title: string;
    description: string;
    confirmText?: string;
    variant: 'danger' | 'warning' | 'default';
    onConfirm: () => void;
  } | null>(null);

  const { container, isLoading, isError } = useDockerData();
  const { restartMutation, stopMutation } = useDockerActions();

  const handleRestart = () => {
    setConfirmConfig({
      isOpen: true,
      title: UI_COPY.restartContainerTitle,
      description: UI_COPY.restartContainerDesc,
      variant: 'warning',
      onConfirm: () => {
        setConfirmConfig(null);
        restartMutation.mutate();
      },
    });
  };

  const handleStop = () => {
    setConfirmConfig({
      isOpen: true,
      title: UI_COPY.stopContainerTitle,
      description: UI_COPY.stopContainerDesc,
      variant: 'danger',
      onConfirm: () => {
        setConfirmConfig(null);
        stopMutation.mutate();
      },
    });
  };

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Container control</p>
        <h1 className="text-2xl font-bold text-base-content">Docker</h1>
      </div>

      <StateContainer
        isLoading={isLoading}
        isError={isError}
        isEmpty={!container}
        emptySlot={<div className="surface-card p-6 text-base-content/60">No container data available</div>}
      >
        {/* Container Status */}
        <div className="surface-card p-6">
          <div className="flex items-center gap-4 mb-4">
           <div className={cn(
               'w-4 h-4 rounded-full',
               container?.status === 'running' ? 'bg-success' : 'bg-error'
             )} />
            <span className="text-lg font-medium capitalize text-base-content">{container?.status || 'unknown'}</span>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="text-base-content/60">Container ID</p>
              <p className="font-mono text-base-content">{container?.id || '--'}</p>
            </div>
            <div>
              <p className="text-base-content/60">Name</p>
              <p className="font-medium text-base-content">{container?.name || '--'}</p>
            </div>
            <div>
              <p className="text-base-content/60">Image</p>
              <p className="font-medium text-base-content">{container?.image || '--'}</p>
            </div>
            <div>
              <p className="text-base-content/60">Created</p>
              <p className="font-medium text-base-content">
                {container?.created ? new Date(container.created).toLocaleString() : '--'}
              </p>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-4">
          <button
            onClick={handleRestart}
            disabled={restartMutation.isPending}
            className="action-btn-primary"
          >
            <RotateCcw className={cn('w-4 h-4', restartMutation.isPending && 'animate-spin')} />
            Restart Container
          </button>
          <button
            onClick={handleStop}
            disabled={stopMutation.isPending || container?.status !== 'running'}
            className="action-btn-danger"
          >
            <Square className="w-4 h-4" />
            Stop Container
          </button>
        </div>

        {/* Ports */}
        <div className="surface-card p-6">
          <h2 className="mb-4 font-medium text-base-content">Port Mappings</h2>
          {container?.ports && container.ports.length > 0 ? (
            <div className="space-y-2">
              {container.ports.map((port, i) => (
                <div key={i} className="flex items-center gap-4 text-sm">
                  <span className="font-mono text-base-content">{port.publicPort || '--'}</span>
                  <span className="text-base-content/50">→</span>
                  <span className="font-mono text-base-content">{port.privatePort}</span>
                  <span className="text-base-content/50">({port.type})</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-base-content/60">No ports exposed</p>
          )}
        </div>

        {/* Resources */}
        <div className="surface-card p-6">
          <h2 className="mb-4 font-medium text-base-content">Resources</h2>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-base-content/60">Memory</p>
              <p className="text-xl font-bold text-base-content">
                {container?.state?.memory ? formatBytes(container.state.memory) : '--'}
              </p>
            </div>
            <div>
              <p className="text-sm text-base-content/60">Restart Count</p>
              <p className="text-xl font-bold text-base-content">{container?.state?.restartCount || 0}</p>
            </div>
          </div>
        </div>
      </StateContainer>

      {/* Confirmation Dialog */}
      {confirmConfig && (
        <ConfirmationDialog
          isOpen={confirmConfig.isOpen}
          title={confirmConfig.title}
          description={confirmConfig.description}
          confirmText={confirmConfig.confirmText}
          variant={confirmConfig.variant}
          isPending={restartMutation.isPending || stopMutation.isPending}
          onConfirm={confirmConfig.onConfirm}
          onCancel={() => setConfirmConfig(null)}
        />
      )}
    </div>
  );
}
