import { useCallback, useEffect, useRef, useState } from 'react';
import { type EnvVar } from '@/features/env-vars/services';
import { Plus, FileStack, Download } from 'lucide-react';
import { EnvVarRow } from '@/features/env-vars/components';
import { EnvVarModal } from '@/features/env-vars/components';
import { BulkImportModal } from '@/features/env-vars/components';
import { DotenvEditor } from '@/features/env-vars/components';
import { ConfirmationDialog, StateContainer } from '@/shared/components';
import { EmptyState, ErrorState } from '@/shared/components/feedback';
import { Button, Table, TabPanel, Tabs } from '@/shared/components/ui';
import { useEnvVarsData, useEnvVarsActions } from '@/features/env-vars/hooks';
import { useAuth } from '@/features/auth/hooks';
import { envService } from '@/features/env-vars/services';
import { useT } from '@/i18n';

export function EnvVarsView() {
  const { t } = useT();
  const { user } = useAuth();

  // Data queries
  const { envVars, isLoading, isError, refetchEnvVars } = useEnvVarsData();

  // Mutations
  const { createMutation, deleteMutation } = useEnvVarsActions();

  // UI state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isBulkOpen, setIsBulkOpen] = useState(false);
  const [editingVar, setEditingVar] = useState<EnvVar | null>(null);
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});
  const [activeTab, setActiveTab] = useState<'table' | 'dotenv'>('table');
  const [nodeRedSyncState, setNodeRedSyncState] = useState<'idle' | 'loading' | 'synced' | 'unavailable' | 'error'>('idle');
  // Server summary or failure reason; the sentence around it comes from the catalog.
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
    setNodeRedSyncMessage('');
    const result = await envService.importFromNodeRed(true);
    if (!isMounted.current) return;

    if (result.status !== 'synced') {
      setNodeRedSyncState(result.status);
      setNodeRedSyncMessage(result.summary);
      return;
    }
    if (result.lines.length > 0) refetchEnvVars();
    setNodeRedSyncState('synced');
    setNodeRedSyncMessage(result.summary);
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
      title: t('env-vars:deleteDialogTitle'),
      description: t('env-vars:deleteDialogDescription', { key }),
      confirmText: key,
      variant: 'danger',
      onConfirm: () => {
        setConfirmConfig(null);
        deleteMutation.mutate(key);
      },
    });
  };

  return (
    <div className="space-y-4">
      <header className="flex flex-col gap-3 border-b border-ds-border-subtle pb-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0 space-y-1">
          <h1 className="text-xl font-bold text-ds-text-primary">{t('env-vars:pageTitle')}</h1>
          <p className="max-w-[75ch] text-pretty text-sm text-ds-text-secondary">{t('env-vars:pageIntro')}</p>
        </div>
        {activeTab === 'table' && (
          <div className="flex shrink-0 flex-wrap gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => void syncFromNodeRed()}
              disabled={nodeRedSyncState === 'loading'}
              data-testid="import-from-node-red-button"
              title={t('env-vars:resyncTitle')}
              className="min-h-11 sm:min-h-9"
            >
              <Download className="h-4 w-4" aria-hidden="true" />
              {nodeRedSyncState === 'loading' ? t('env-vars:resyncing') : t('env-vars:resyncNodeRed')}
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsBulkOpen(true)}
              data-testid="bulk-import-button"
              className="min-h-11 sm:min-h-9"
            >
              <FileStack className="h-4 w-4" aria-hidden="true" />
              {t('env-vars:bulkImport')}
            </Button>
            <Button type="button" onClick={() => openModal()} className="min-h-11 sm:min-h-9">
              <Plus className="h-4 w-4" aria-hidden="true" />
              {t('env-vars:add')}
            </Button>
          </div>
        )}
      </header>

      {nodeRedSyncState !== 'idle' && (
        <div
          role="status"
          aria-live="polite"
          data-testid="node-red-sync-status"
          className={`rounded-box border px-3 py-2 text-sm ${
            nodeRedSyncState === 'error'
              ? 'border-ds-danger/35 bg-ds-danger/10 text-ds-on-tint-error'
              : nodeRedSyncState === 'unavailable'
                ? 'border-ds-warning/35 bg-ds-warning/10 text-ds-on-tint-warning'
                : 'border-ds-border-subtle text-ds-text-secondary'
          }`}
        >
          {nodeRedSyncState === 'loading'
            ? t('env-vars:syncLoading')
            : nodeRedSyncState === 'unavailable'
              ? t('env-vars:syncUnavailable', { reason: nodeRedSyncMessage })
              : nodeRedSyncState === 'error'
                ? t('env-vars:syncError', { reason: nodeRedSyncMessage })
                : nodeRedSyncMessage
                  ? t('env-vars:syncSynced', { summary: nodeRedSyncMessage })
                  : t('env-vars:syncSyncedEmpty')}
        </div>
      )}

      <Tabs
        label={t('env-vars:tabsLabel')}
        idBase="env-vars"
        items={[
          { id: 'table', label: t('env-vars:tabConfigured') },
          { id: 'dotenv', label: t('env-vars:tabDotenv') },
        ]}
        selected={activeTab}
        onSelect={setActiveTab}
      />

      <TabPanel idBase="env-vars" selected={activeTab}>
        {activeTab === 'table' ? (
          <StateContainer
            isLoading={isLoading}
            isError={!!isError}
            isEmpty={envVars.length === 0}
            errorSlot={<ErrorState title={t('env-vars:loadError')} onRetry={() => void refetchEnvVars()} />}
            emptySlot={<EmptyState title={t('env-vars:noEnvVarsConfigured')} description={t('env-vars:noEnvVarsHint')} />}
          >
            <Table stacked caption={t('env-vars:pageTitle')}>
              <thead>
                <tr>
                  <th>{t('env-vars:colKey')}</th>
                  <th>{t('env-vars:colValue')}</th>
                  <th>{t('env-vars:colType')}</th>
                  <th>{t('env-vars:colSource')}</th>
                  <th>{t('env-vars:colDescription')}</th>
                  <th className="text-right">{t('env-vars:colActions')}</th>
                </tr>
              </thead>
              <tbody>
                {envVars.map((envVar) => (
                  <EnvVarRow
                    key={envVar.key}
                    envVar={envVar}
                    onDelete={handleDelete}
                    onEdit={openModal}
                    onToggleSecret={toggleSecret}
                    showSecret={!!showSecrets[envVar.key]}
                  />
                ))}
              </tbody>
            </Table>
          </StateContainer>
        ) : (
          <DotenvEditor />
        )}
      </TabPanel>

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
