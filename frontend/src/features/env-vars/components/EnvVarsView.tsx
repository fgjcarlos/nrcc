import { useCallback, useEffect, useRef, useState } from 'react';
import { type EnvVar } from '@/features/env-vars/services';
import { Plus, FileStack, Download } from 'lucide-react';
import { EnvVarRow } from '@/features/env-vars/components';
import { EnvVarModal } from '@/features/env-vars/components';
import { BulkImportModal } from '@/features/env-vars/components';
import { DotenvEditor } from '@/features/env-vars/components';
import { ConfirmationDialog } from '@/shared/components';
import { useEnvVarsData, useEnvVarsActions } from '@/features/env-vars/hooks';
import { useAuth } from '@/features/auth/hooks';
import { envService } from '@/features/env-vars/services';

export function EnvVarsView() {
  const { user } = useAuth();

  // Data queries
  const { envVars, isLoading, refetchEnvVars } = useEnvVarsData();

  // Mutations
  const { createMutation, deleteMutation } = useEnvVarsActions();

  // UI state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isBulkOpen, setIsBulkOpen] = useState(false);
  const [editingVar, setEditingVar] = useState<EnvVar | null>(null);
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});
  const [activeTab, setActiveTab] = useState<'table' | 'dotenv'>('table');
  const [nodeRedSyncState, setNodeRedSyncState] = useState<'idle' | 'loading' | 'synced' | 'unavailable' | 'error'>('idle');
  const [nodeRedSyncMessage, setNodeRedSyncMessage] = useState('');
  const didAutoImport = useRef(false);
  const isMounted = useRef(false);
  const [confirmConfig, setConfirmConfig] = useState<{
    isOpen: boolean;
    title: string;
    description: string;
    confirmText?: string;
    variant: 'danger' | 'warning' | 'default';
    onConfirm: () => void;
  } | null>(null);
  const [formData, setFormData] = useState({
    key: '',
    value: '',
    type: 'string' as EnvVar['type'],
    description: '',
  });

  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
    };
  }, []);

  const syncFromNodeRed = useCallback(async () => {
    setNodeRedSyncState('loading');
    setNodeRedSyncMessage('Synchronizing Node-RED environment entries…');
    const result = await envService.importFromNodeRed(true);
    if (!isMounted.current) return;

    if (result.status !== 'synced') {
      setNodeRedSyncState(result.status);
      setNodeRedSyncMessage(result.summary);
      return;
    }
    if (result.lines.length > 0) refetchEnvVars();
    setNodeRedSyncState('synced');
    setNodeRedSyncMessage(result.summary ? `Node-RED synchronized: ${result.summary}` : 'Node-RED environment entries are synchronized.');
  }, [refetchEnvVars]);

  useEffect(() => {
    if (user?.role !== 'admin' || didAutoImport.current) return;
    didAutoImport.current = true;
    void syncFromNodeRed();
  }, [syncFromNodeRed, user?.role]);

  const openModal = (envVar?: EnvVar) => {
    if (envVar) {
      setEditingVar(envVar);
      // Only blank the value for encrypted/secret types; for others, pre-populate with actual value
      const value = envVar.encrypted ? '' : envVar.value;
      setFormData({
        key: envVar.key,
        value: value,
        type: envVar.type,
        description: envVar.description || '',
      });
    } else {
      setEditingVar(null);
      setFormData({ key: '', value: '', type: 'string', description: '' });
    }
    setIsModalOpen(true);
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setEditingVar(null);
    setFormData({ key: '', value: '', type: 'string', description: '' });
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    createMutation.mutate(formData, {
      onSuccess: () => {
        closeModal();
      },
    });
  };

  const toggleSecret = (key: string) => {
    setShowSecrets(prev => ({ ...prev, [key]: !prev[key] }));
  };

  const handleDelete = (key: string) => {
    setConfirmConfig({
      isOpen: true,
      title: 'Delete environment variable',
      description: `Are you sure you want to delete ${key}? Node-RED will restart.`,
      confirmText: key,
      variant: 'danger',
      onConfirm: () => {
        setConfirmConfig(null);
        deleteMutation.mutate(key);
      },
    });
  };

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-between items-center">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Secrets</p>
          <h1 className="text-2xl font-bold text-base-content">Environment Variables</h1>
        </div>
        {activeTab === 'table' && (
          <div className="flex gap-2">
            <button
              onClick={() => void syncFromNodeRed()}
              disabled={nodeRedSyncState === 'loading'}
              className="action-btn-secondary disabled:cursor-not-allowed disabled:opacity-60"
              data-testid="import-from-node-red-button"
              title="Resynchronize environment entries from Node-RED 5 global-config"
            >
              <Download className="w-4 h-4" />
              {nodeRedSyncState === 'loading' ? 'Resyncing…' : 'Resync Node-RED'}
            </button>
            <button
              onClick={() => setIsBulkOpen(true)}
              className="action-btn-secondary"
              data-testid="bulk-import-button"
            >
              <FileStack className="w-4 h-4" />
              Bulk import
            </button>
            <button onClick={() => openModal()} className="action-btn-primary">
              <Plus className="w-4 h-4" />
              Add
            </button>
          </div>
        )}
      </div>

      {nodeRedSyncState !== 'idle' && (
        <div
          role="status"
          aria-live="polite"
          data-testid="node-red-sync-status"
          className={`rounded-lg border px-4 py-3 text-sm ${
            nodeRedSyncState === 'error'
              ? 'border-error/30 bg-error/10 text-error-content'
              : nodeRedSyncState === 'unavailable'
                ? 'border-warning/30 bg-warning/10 text-warning-content'
                : 'border-info/30 bg-info/10 text-base-content'
          }`}
        >
          {nodeRedSyncState === 'unavailable'
            ? `Node-RED synchronization is unavailable: ${nodeRedSyncMessage}`
            : nodeRedSyncState === 'error'
              ? `Could not synchronize Node-RED environment entries: ${nodeRedSyncMessage}`
              : nodeRedSyncMessage}
        </div>
      )}

      {/* TAREA 3: Tab Navigation */}
      <div className="tabs tabs-bordered">
        <button
          onClick={() => setActiveTab('table')}
          className={`tab ${activeTab === 'table' ? 'tab-active' : ''}`}
        >
          Configured
        </button>
        <button
          onClick={() => setActiveTab('dotenv')}
          className={`tab ${activeTab === 'dotenv' ? 'tab-active' : ''}`}
        >
          .env file
        </button>
      </div>

      {/* Variables Table Tab */}
      {activeTab === 'table' && (
        <div className="surface-card overflow-hidden">
          <table className="w-full">
            <thead className="table-header-subtle">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-base-content">Key</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-base-content">Value</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-base-content">Type</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-base-content">Source</th>
                <th className="px-4 py-3 text-left text-sm font-medium text-base-content">Description</th>
                <th className="px-4 py-3 text-right text-sm font-medium text-base-content">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-base-content/60">
                    Loading...
                  </td>
                </tr>
                ) : envVars.length === 0 ? (
                  <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-base-content/60">
                      No environment variables configured
                    </td>
                  </tr>
                ) : (
                  envVars.map((envVar) => (
                    <EnvVarRow
                      key={envVar.key}
                      envVar={envVar}
                      onDelete={handleDelete}
                      onEdit={openModal}
                      onToggleSecret={toggleSecret}
                      showSecret={!!showSecrets[envVar.key]}
                    />
                  ))
                )}
            </tbody>
          </table>
        </div>
      )}

      {/* .env Editor Tab - TAREA 3 */}
      {activeTab === 'dotenv' && <DotenvEditor />}

       {/* Modal */}
      {isModalOpen && (
        <EnvVarModal
          editing={!!editingVar}
          formData={formData}
          setFormData={setFormData}
          onCancel={closeModal}
          onSubmit={handleSubmit}
          isPending={createMutation.isPending}
        />
      )}
      <BulkImportModal
        open={isBulkOpen}
        onClose={() => setIsBulkOpen(false)}
        onImported={() => refetchEnvVars()}
      />

      {confirmConfig && (
        <ConfirmationDialog
          isOpen={confirmConfig.isOpen}
          title={confirmConfig.title}
          description={confirmConfig.description}
          confirmText={confirmConfig.confirmText}
          variant={confirmConfig.variant}
          isPending={deleteMutation.isPending}
          onConfirm={confirmConfig.onConfirm}
          onCancel={() => setConfirmConfig(null)}
        />
      )}
    </div>
  );
}
