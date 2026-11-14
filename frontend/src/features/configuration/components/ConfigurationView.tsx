import { useState, useEffect, useId, useMemo, useRef } from 'react';
import { toast } from 'sonner';
import { cn } from '@/shared/lib';
import type { NodeRedConfigFormData } from '@/shared/types';
import { settingsRawSchema } from '@/shared/validation/schemas';
import {
  BasicSettings,
  SecuritySettings,
  LoggingSettings,
  EditorThemeSettings,
  AIProviderSettings,
} from '.';
import { AdvancedSettings } from './AdvancedSettings';
import { ConfigurationHeader } from './ConfigurationHeader';
import { ReviewChangesPanel } from './ReviewChangesPanel';
import { ChangeCircuit } from './ChangeCircuit';
import { applyErrorCode, deriveCircuit, type ApplyRun } from '../lib/changeCircuit';
import {
  Server, Shield, Activity, Palette,
  Save, LockOpen, AlertTriangle, Bot
} from 'lucide-react';
import { useConfigurationData, useConfigurationActions, useConfigurationDiff, useConfigurationSave, fieldValidationMap } from '../hooks';
import { useT } from '@/i18n';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { LoadingBoundary } from '@/shared/components/feedback';

// ============================================
// Sections Configuration
// ============================================

interface Section {
  id: string;
  label: string;
  icon: React.ElementType;
  component?: React.ComponentType<{
    settings: NodeRedConfigFormData;
    onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
    disabled?: boolean;
  }>;
}

const SECTIONS: Section[] = [
  { id: 'basic', label: 'Basic', icon: Server, component: BasicSettings },
  // Issue #766 slice A — auth surfaces moved to /security.
  // Issue #762 — TLS, credentialSecret and requireHttps tab.
  { id: 'tls', label: 'TLS & Secrets', icon: Shield, component: SecuritySettings },
  { id: 'logging', label: 'Logging', icon: Activity, component: LoggingSettings },
  { id: 'editor', label: 'Editor Theme', icon: Palette, component: EditorThemeSettings },
  { id: 'ai', label: 'AI Provider', icon: Bot },
];

// ============================================
// Initial Form State
// ============================================

const INITIAL_FORM_DATA: NodeRedConfigFormData = {
  // Basic
  uiPort: 1880,
  uiHost: '0.0.0.0',
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  disableEditor: false,

  // Authentication

  // Projects
  projectsEnabled: false,

  // Logging - Multiple handlers
  loggingConsoleLevel: 'info',
  loggingConsoleMetrics: false,
  loggingInternalLevel: 'info',
  loggingInternalMetrics: false,

  // Files
  flowFile: 'flows.json',
  userDir: '',
  nodesDir: '',

  // Editor - Page
  editorPageTitle: 'Node-RED',
  editorPageFavicon: '',
  editorPageCss: '',

  // Editor - Header
  editorHeaderTitle: 'Node-RED',
  editorHeaderImage: '',
  editorHeaderUrl: '',

  // Editor - Deploy
  editorDeployType: 'default',
  editorDeployLabel: 'Deploy',
  editorDeployIcon: '',

  // Editor - Palette
  editorPaletteEditable: true,
  editorPaletteCatalogues: '',

  // Editor - Projects
  editorProjectsEnabled: false,

  // Editor - Code
  editorCodeLib: 'ace',
  editorCodeTheme: 'vs',
  editorCodeFontSize: 12,

  // Editor - Misc
  editorUserMenu: true,
  editorTours: true,

  // Editor - Login
  editorLoginImage: '',

  // Editor - Logout
  editorLogoutRedirect: '',

  // Runtime State
  runtimeStateEnabled: false,
  runtimeStateFile: '',

  // Language
  lang: 'en-US',
};

// ============================================
// Main Configuration Page
// ============================================

export function ConfigurationView() {
  // UI state
  const [activeTab, setActiveTab] = useState('basic');
  const [formData, setFormData] = useState<NodeRedConfigFormData>(INITIAL_FORM_DATA);
  const [rawSettingsContent, setRawSettingsContent] = useState('');
  // Raw-settings editor gate (issue #364): the textarea is read-only by
  // default. The user must click "Unlock to edit settings.js", acknowledge
  // the warning dialog, and only then can they edit and save. Re-locking
  // discards any in-flight edits.
  const [rawEditorUnlocked, setRawEditorUnlocked] = useState(false);
  const [rawEditorSnapshot, setRawEditorSnapshot] = useState('');
  const [unlockDialogOpen, setUnlockDialogOpen] = useState(false);

  // Issue #762 — credentialSecret rotation confirmation (handleApplyFromPanel
  // gates when the operator typed a non-empty secret).
  const [rotationDialogOpen, setRotationDialogOpen] = useState(false);

  // Issue #766 slice F (W4) — reviewable safe-apply panel. Opens from
  // the change circuit's "Review N changes" action. The actual save
  // still goes through `runSave` so the credentialSecret rotation dialog
  // and the raw-settings unlock flow remain the single source of truth.
  const [reviewPanelOpen, setReviewPanelOpen] = useState(false);
  // T9 change circuit: the form state the operator last reviewed (the
  // Diff contact) and the outcome of the last server transaction.
  const [reviewedFormData, setReviewedFormData] = useState<NodeRedConfigFormData | null>(null);
  const [applyRun, setApplyRun] = useState<ApplyRun>({ status: 'idle' });
  // Apply in the drawer asks for confirmation before saving.
  const [applyConfirmOpen, setApplyConfirmOpen] = useState(false);
  // Synchronous guard: the mutation's pending flag lands after a render,
  // so a same-tick double click would otherwise start two saves.
  const savingRef = useRef(false);

  // Data and actions hooks
  const { t } = useT();
  // Names the raw settings.js editor after its section heading (#889 axe `label`).
  const rawSettingsHeadingId = useId();

  const data = useConfigurationData();
  const actions = useConfigurationActions();

  // Issue #766 slice F — derive per-field diff once, share with the
  // save gate and the per-field `FieldStatusChip` widgets inside the
  // active section component.
  const loadedConfig = data.config;
  const hostStatus = data.hostStatus ?? null;
  const diff = useConfigurationDiff(
    formData,
    loadedConfig,
    hostStatus,
    data.rawSettingsContent,
  );
  // Issue #878 — the save travels with the settings.js revision the
  // operator read. The backend rejects a stale revision with 409
  // SETTINGS_REVISION_CONFLICT instead of overwriting newer state, so
  // two browser sessions cannot silently clobber each other.
  const expectedRevision = data.settingsDoc?.revision?.fingerprint;
  const save = useConfigurationSave({ formData, diff, expectedRevision });

  // Sync loaded config to form
  useEffect(() => {
    if (data.initialFormData) {
      setFormData(data.initialFormData);
    }
  }, [data.initialFormData]);

  // Sync raw settings content
  useEffect(() => {
    if (data.rawSettingsContent) {
      setRawSettingsContent(data.rawSettingsContent);
    }
  }, [data.rawSettingsContent]);

  // Handlers
  const handleUpdateField = (
    field: keyof NodeRedConfigFormData,
    value: string | number | boolean
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    // A new edit starts a new circuit; the last outcome no longer applies.
    setApplyRun((run) => (run.status === 'running' ? run : { status: 'idle' }));
  };

  const handleReview = () => {
    // Issue #766 slice F (W4) — reviewable safe-apply flow. The panel's
    // Apply button drives the actual save via `handleApplyFromPanel`.
    setReviewedFormData(formData);
    setReviewPanelOpen(true);
  };

  // Runs the save once at a time; null means a save is already running.
  const runSave = async () => {
    if (savingRef.current) return null;
    savingRef.current = true;
    const count = diff.pendingCount;
    setApplyRun({ status: 'running', count });
    try {
      const outcome = await save.save();
      setApplyRun(
        outcome.ok
          ? { status: 'succeeded', count }
          : outcome.reason === 'mutation-failed'
            ? { status: 'failed', count, code: applyErrorCode(outcome.error) }
            : { status: 'idle' },
      );
      return outcome;
    } finally {
      savingRef.current = false;
    }
  };

  const handleApplyFromPanel = () => {
    // Issue #762 — rotation confirmation gates the save on a new
    // credentialSecret; it already confirms, so no second dialog.
    if (formData.credentialSecret) {
      setReviewPanelOpen(false);
      setRotationDialogOpen(true);
      return;
    }
    setApplyConfirmOpen(true);
  };

  const handleConfirmApply = async () => {
    const outcome = await runSave();
    if (!outcome) return;
    setApplyConfirmOpen(false);
    if (outcome.ok) {
      setReviewPanelOpen(false);
    } else if (outcome.reason === 'validation-errors') {
      toast.error(`Fix ${Object.keys(outcome.errors).length} field${Object.keys(outcome.errors).length === 1 ? '' : 's'} before saving`);
    } else if (outcome.reason === 'no-pending-changes') {
      // Operator applied without changing anything; treat as a
      // soft no-op.
      setReviewPanelOpen(false);
    } else if (outcome.reason === 'read-only') {
      toast.error('Configuration is read-only on this host');
    }
  };

  const handleConfirmRotation = async () => {
    // Keep the dialog open while saving so it shows its processing state.
    const outcome = await runSave();
    if (!outcome) return;
    setRotationDialogOpen(false);
    if (outcome.ok) return;
    if (outcome.reason === 'validation-errors') {
      toast.error(`Fix ${Object.keys(outcome.errors).length} field${Object.keys(outcome.errors).length === 1 ? '' : 's'} before saving`);
    } else if (outcome.reason === 'read-only') {
      toast.error('Configuration is read-only on this host');
    }
  };

  const handleCancelRotation = () => {
    setRotationDialogOpen(false);
    setFormData((prev) => ({ ...prev, credentialSecret: undefined }));
  };

  // Discard returns to the loaded configuration, not to the form defaults,
  // so the circuit drops back to zero pending changes.
  const handleReset = () => {
    setFormData(data.initialFormData ?? INITIAL_FORM_DATA);
    setApplyRun({ status: 'idle' });
  };

  // Issue #364: locking the editor discards any in-flight edits and
  // restores the snapshot we took when the user clicked "Unlock".
  const handleCancelRawEdit = () => {
    setRawSettingsContent(rawEditorSnapshot);
    setRawEditorUnlocked(false);
  };

  // Called when the user clicks the "Save changes" button on the unlocked
  // textarea. After a successful save we re-lock the editor so the next
  // session starts in the safe default.
  const handleSaveRawSettingsLocked = (content: string) => {
    const result = settingsRawSchema.safeParse(content);
    if (!result.success) {
      toast.error(`Invalid settings.js: ${result.error.issues[0]?.message ?? 'syntax error'}`);
      return;
    }
    // Issue #878 — the raw editor sends the revision of the document it
    // opened, so a stale editor session cannot overwrite newer source.
    actions.handleSaveRawSettings(content, expectedRevision);
    // Snapshot the saved content so a re-lock restores the saved state,
    // not the pre-edit state.
    setRawEditorSnapshot(content);
  };

  // Derived state
  const ActiveComponent = SECTIONS.find(s => s.id === activeTab)?.component || BasicSettings;
  // `save` owns the mutation instance that actually runs the request.
  const isSaving = save.isPending;
  const readOnly = data.hostStatus?.configuration?.editable === false;
  const circuit = useMemo(
    () =>
      deriveCircuit({
        pendingCount: diff.pendingCount,
        invalidCount: Object.keys(diff.validationErrors).length,
        reviewed: reviewedFormData === formData,
        readOnly,
        run: applyRun,
      }),
    [diff, reviewedFormData, formData, readOnly, applyRun],
  );
  const restartCount = Object.values(diff.fields).filter((f) => f.pending && f.restart === 'hard').length;
  const host = data.hostStatus as (typeof data.hostStatus & { portless?: { installed?: boolean } }) | undefined;
  const recommendations = host
    ? [
        host.nodeRed.detected && host.portless?.installed === false ? t('configuration:host.recommendPortless') : null,
        host.settings?.path && host.settings.writable === false ? t('configuration:host.recommendWritable') : null,
      ].filter((r): r is string => r !== null)
    : [];
  const configViewState = data.configLoading
    ? { status: 'pending' as const }
    : data.configError
      ? {
          status: 'error' as const,
          error: data.configError,
          onRetry: () => {
            void data.refetchConfig();
          },
        }
      : { status: 'success' as const };

  return (
    <LoadingBoundary state={configViewState}>
    <div className="space-y-4">
      {/* Header (slice F W3, T9) — title, one runtime status, host details, topology */}
      <ConfigurationHeader
        status={
          data.hostStatus && (
            <p className="text-sm text-ds-text-secondary" data-testid="configuration-runtime-status">
              <span className="font-mono tabular-nums">
                {`Node-RED ${data.hostStatus.configuration?.runtimeVersion || data.hostStatus.nodeRed.version || t('common:unknown')}`}
              </span>
              {' · '}
              <strong className={cn('font-semibold', readOnly ? 'text-ds-on-tint-warning' : 'text-ds-on-tint-success')}>
                {readOnly ? t('configuration:readOnly') : t('configuration:editable')}
              </strong>
            </p>
          )
        }
      >
        {data.hostStatus && (
          <dl className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-ds-text-secondary">
            <div className="flex gap-1">
              <dt>{t('configuration:host.installationDetected')}:</dt>
              <dd className="font-semibold text-ds-text-primary">
                {data.hostStatus.nodeRed.mode}
                {data.hostStatus.nodeRed.detected ? '' : ` ${t('configuration:host.nodeRedNotDetected')}`}
              </dd>
            </div>
            <div className="flex gap-1">
              <dt>{t('configuration:configSourceLabel')}:</dt>
              <dd className="font-semibold text-ds-text-primary">
                {data.hostStatus.configuration?.source || data.hostStatus.settings.source}
              </dd>
            </div>
            {data.hostStatus.configuration?.reason && (
              <div className="flex gap-1">
                <dt>{t('configuration:editingLabel')}:</dt>
                <dd>{data.hostStatus.configuration.reason}</dd>
              </div>
            )}
          </dl>
        )}
        {recommendations.map((recommendation) => (
          <p key={recommendation} className="text-xs text-ds-text-secondary">{recommendation}</p>
        ))}
      </ConfigurationHeader>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_20rem] lg:gap-6 xl:grid-cols-[minmax(0,1fr)_21rem]">
      {/* T9 change circuit: first in reading order so it sits above the
          form as a strip below lg; a sticky right column from lg up. */}
      <ChangeCircuit
        className="lg:col-start-2 lg:row-start-1"
        circuit={circuit}
        pendingCount={diff.pendingCount}
        restartCount={restartCount}
        reviewDisabled={!diff.canSave || isSaving}
        onReview={handleReview}
        onDiscard={diff.pendingCount > 0 && !isSaving ? handleReset : undefined}
      />

      <div className="min-w-0 space-y-4 lg:col-start-1 lg:row-start-1">
      {/* Tabs Navigation */}
      <div className="border-b ghost-divider">
        <nav className="flex space-x-1 overflow-x-auto" aria-label="Tabs">
          {SECTIONS.map((section) => {
            const Icon = section.icon;
            return (
              <button
                key={section.id}
                onClick={() => setActiveTab(section.id)}
                className={cn(
                  'flex items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors',
                  activeTab === section.id
                    ? 'border-primary text-primary'
                    : 'border-transparent text-base-content/55 hover:border-border hover:text-base-content'
                )}
              >
                <Icon className="w-4 h-4" />
                {section.label}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Active Tab Content */}
      <div>
        {readOnly ? (
          <div className="rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm text-warning" role="status">
            {t('configuration:controlsUnavailable')}
          </div>
        ) : activeTab === 'ai' ? <AIProviderSettings /> : <ActiveComponent settings={formData} onUpdate={handleUpdateField} disabled={isSaving} fieldErrors={fieldValidationMap(diff.fields)} />}
      </div>

      {/* Curated preset catalogue — slice 3 of #764 */}
      <AdvancedSettings
        rawContent={rawSettingsContent}
        expectedRevision={expectedRevision}
        onApplied={() => { void data.refetchConfig(); void data.refetchSettings(); }}
      />

      {/* Raw Settings Editor — gated by issue #364 */}
      <div className="space-y-4 border-t border-ds-border-subtle pt-4">
        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1">
          <div className="min-w-0">
            <h2 id={rawSettingsHeadingId} className="text-lg font-semibold text-base-content">
              {t('configuration:advanced.title')}
            </h2>
            <p className="text-sm text-base-content/65">
              {t('configuration:advanced.description')}
            </p>
          </div>
          {data.settingsDoc?.backupPath && (
            <span className="min-w-0 break-all text-xs text-base-content/55">
              {t('configuration:advanced.lastBackup', { path: data.settingsDoc.backupPath })}
            </span>
          )}
        </div>

        {!rawEditorUnlocked && (
          <div
            data-testid="raw-settings-locked-banner"
            className="flex items-center gap-2 rounded-xl border border-warning/40 bg-warning/10 px-3 py-2 text-sm text-warning"
          >
            <AlertTriangle className="h-4 w-4 flex-shrink-0" />
            <span>{t('configuration:advanced.lockedBadge')}</span>
          </div>
        )}

        <textarea
          value={rawSettingsContent}
          onChange={(event) => {
            if (!rawEditorUnlocked) return;
            setRawSettingsContent(event.target.value);
          }}
          readOnly={!rawEditorUnlocked}
          aria-readonly={!rawEditorUnlocked}
          aria-labelledby={rawSettingsHeadingId}
          className={cn(
            'min-h-[20rem] w-full rounded-2xl border border-border bg-base-100/80 p-4 font-mono text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary',
            !rawEditorUnlocked && 'cursor-not-allowed opacity-70'
          )}
          spellCheck={false}
        />

        <div className="flex justify-end gap-2">
          {!rawEditorUnlocked ? (
            <button
              type="button"
              onClick={() => setUnlockDialogOpen(true)}
              disabled={!data.settingsDoc?.writable || actions.saveRawSettingsMutation.isPending}
              data-testid="raw-settings-unlock-btn"
              className="action-btn-secondary"
            >
              <LockOpen className="w-4 h-4" />
              {t('configuration:advanced.unlockToEdit')}
            </button>
          ) : (
            <>
              <button
                type="button"
                onClick={handleCancelRawEdit}
                disabled={actions.saveRawSettingsMutation.isPending}
                data-testid="raw-settings-cancel-btn"
                className="action-btn-secondary"
              >
                {t('configuration:advanced.cancelEdit')}
              </button>
              <button
                type="button"
                onClick={() => handleSaveRawSettingsLocked(rawSettingsContent)}
                disabled={!data.settingsDoc?.writable || actions.saveRawSettingsMutation.isPending}
                data-testid="raw-settings-save-btn"
                className="action-btn-primary"
              >
                <Save className="w-4 h-4" />
                {actions.saveRawSettingsMutation.isPending ? t('configuration:advanced.savingRaw') : t('common:saveChanges')}
              </button>
            </>
          )}
        </div>
      </div>

      </div>
      </div>

      <ConfirmationDialog
        isOpen={unlockDialogOpen}
        title={t('configuration:advanced.unlockDialogTitle')}
        description={t('configuration:advanced.unlockDialogDescription', { backupPath: data.settingsDoc?.backupPath || '/var/backups/nrcc/settings.js' })}
        acknowledgement={t('configuration:advanced.unlockAcknowledgement')}
        variant="warning"
        onCancel={() => setUnlockDialogOpen(false)}
        onConfirm={() => {
          // Snapshot the content on unlock so a Cancel re-lock discards
          // in-flight edits.
          setRawEditorSnapshot(rawSettingsContent);
          setRawEditorUnlocked(true);
          setUnlockDialogOpen(false);
        }}
      />

      {/* Issue #762 — credentialSecret rotation confirmation. */}
      <ConfirmationDialog
        isOpen={rotationDialogOpen}
        title={t('configuration:rotateSecretTitle')}
        description={t('configuration:rotateSecretDescription')}
        variant="warning"
        acknowledgement={t('configuration:rotateSecretAcknowledgement')}
        isPending={isSaving}
        onCancel={handleCancelRotation}
        onConfirm={handleConfirmRotation}
      />

      {/* Issue #766 slice F (W4) — reviewable safe-apply drawer. */}
      <ReviewChangesPanel
        isOpen={reviewPanelOpen}
        fields={diff.fields}
        pendingCount={diff.pendingCount}
        target="settings.js"
        isPending={isSaving}
        onApply={handleApplyFromPanel}
        onCancel={() => setReviewPanelOpen(false)}
      />

      <ConfirmationDialog
        isOpen={applyConfirmOpen}
        title={t('configuration:reviewPanel.confirmTitle')}
        description={t('configuration:reviewPanel.confirmDescription', { count: diff.pendingCount, target: 'settings.js' })}
        variant="warning"
        isPending={isSaving}
        onCancel={() => setApplyConfirmOpen(false)}
        onConfirm={handleConfirmApply}
      />
    </div>
    </LoadingBoundary>
  );
}
