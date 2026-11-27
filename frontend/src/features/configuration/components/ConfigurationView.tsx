import { useState, useEffect, useId, useMemo, useRef } from 'react';
import { useLocation } from 'react-router-dom';
import { errorMessage } from '@/shared/lib/errorMessage';
import { cn } from '@/shared/lib';
import type { NodeRedConfigFormData } from '@/shared/types';
import { DEFAULT_HTTP_NODE_CORS_METHODS } from '@/shared/types';
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
import { SETTINGS_KEYS } from '../lib/fieldLabels';

/** Form field names to the settings.js keys shown as circuit designators. */
const settingsKeys = (fields: string[]) =>
  fields.map((f) => SETTINGS_KEYS[f as keyof NodeRedConfigFormData] ?? f);

import {
  Server, Shield, Activity, Palette,
  Save, Lock, LockOpen, Bot, ChevronDown
} from 'lucide-react';
import { useConfigurationData, useConfigurationActions, useConfigurationDiff, useConfigurationSave, fieldValidationMap } from '../hooks';
import { useT } from '@/i18n';
import type { HostStatusWithCodes } from '@/features/bootstrap/types';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';
import { LoadingBoundary } from '@/shared/components/feedback';
import { TabPanel, Tabs } from '@/shared/components/ui';

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
    configuredConsoleLevel?: string;
    runningConsoleLevel?: string;
    runtimeConsoleLoggingStatus?: 'applied' | 'pending' | 'unknown';
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
  loggingConsoleAudit: undefined,
  loggingInternalLevel: 'info',
  loggingInternalMetrics: false,
  loggingInternalAudit: undefined,

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

  // CORS (issue #1076) — mirrors the defaults `configToFormData` produces
  // for an absent `httpNodeCors` key.
  httpNodeCorsEnabled: false,
  httpNodeCorsOrigins: '',
  httpNodeCorsMethods: DEFAULT_HTTP_NODE_CORS_METHODS.join(','),
  httpNodeCorsCredentials: false,
  httpNodeCorsAllowedHeaders: '',
  httpNodeCorsUnsupported: false,
  httpNodeCorsRaw: '',
};

// ============================================
// Main Configuration Page
// ============================================

export function ConfigurationView() {
  // UI state
  const location = useLocation();
  const [activeTab, setActiveTab] = useState(location.hash === '#ai' ? 'ai' : 'basic');

  useEffect(() => {
    if (location.hash === '#ai') setActiveTab('ai');
  }, [location.hash]);
  const [formData, setFormData] = useState<NodeRedConfigFormData>(INITIAL_FORM_DATA);
  const [rawSettingsContent, setRawSettingsContent] = useState('');
  // Raw-settings editor gate (issue #364): the textarea is read-only by
  // default. The user must click "Unlock to edit settings.js", acknowledge
  // the warning dialog, and only then can they edit and save. Re-locking
  // discards any in-flight edits.
  const [rawEditorUnlocked, setRawEditorUnlocked] = useState(false);
  const [rawParseError, setRawParseError] = useState<string | null>(null);
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
  // Synchronous guard: the mutation's pending flag lands after a render,
  // so a same-tick double click would otherwise start two saves.
  const savingRef = useRef(false);

  // Data and actions hooks
  const { t, i18n } = useT();
  const rawErrorId = useId();
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
            // Gated outcomes stay on the circuit, not on a toast (THESIS).
            : outcome.reason === 'validation-errors'
              ? { status: 'held', count, reason: 'invalid', fields: settingsKeys(Object.keys(outcome.errors)) }
              : outcome.reason === 'read-only'
                ? { status: 'held', count, reason: 'readOnly' }
                : { status: 'idle' },
      );
      return outcome;
    } finally {
      savingRef.current = false;
    }
  };

  // The review drawer is the single confirmation: its Apply saves directly.
  const handleApplyFromPanel = async () => {
    // Issue #762 — rotation confirmation gates the save on a new
    // credentialSecret with its own acknowledgement.
    if (formData.credentialSecret) {
      setReviewPanelOpen(false);
      setRotationDialogOpen(true);
      return;
    }
    const outcome = await runSave();
    if (!outcome) return;
    // Every settled outcome closes the drawer: the circuit beside it carries
    // the result (closed contacts, a held gate or the rollback branch), and
    // the drawer would otherwise cover it. Review reopens it for a retry.
    setReviewPanelOpen(false);
  };

  const handleConfirmRotation = async () => {
    // Keep the dialog open while saving so it shows its processing state.
    const outcome = await runSave();
    if (!outcome) return;
    // The circuit carries every outcome of the rotation save.
    setRotationDialogOpen(false);
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
    setRawParseError(null);
  };

  // Called when the user clicks the "Save changes" button on the unlocked
  // textarea. After a successful save we re-lock the editor so the next
  // session starts in the safe default.
  const handleSaveRawSettingsLocked = (content: string) => {
    const result = settingsRawSchema.safeParse(content);
    if (!result.success) {
      setRawParseError(result.error.issues[0]?.message ?? 'syntax error');
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
        invalidFields: settingsKeys(Object.keys(diff.validationErrors)),
        reviewed: reviewedFormData === formData,
        readOnly,
        run: applyRun,
      }),
    [diff, reviewedFormData, formData, readOnly, applyRun],
  );
  // Inline errors under the raw editor: client syntax check, then the server.
  const rawError = rawParseError
    ? t('configuration:advanced.parseError', { message: rawParseError })
    : actions.saveRawSettingsMutation.error
      ? errorMessage(actions.saveRawSettingsMutation.error)
      : null;
  const revision = data.settingsDoc?.revision?.fingerprint;
  const restartCount = Object.values(diff.fields).filter((f) => f.pending && f.restart === 'hard').length;
  // Localize each backend recommendation by its stable code (shared with Host & environment); unknown codes keep the backend text.
  const host = data.hostStatus as HostStatusWithCodes | undefined;
  const recommendations = (host?.recommendations ?? []).map((message, index) => {
    const key = `bootstrap:recommendation.${host?.recommendationCodes?.[index]}`;
    return host?.recommendationCodes?.[index] && i18n.exists(key) ? t(key) : message;
  });
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
      {/* Header (slice F W3, T9) — title, revision designator, one runtime status, host details */}
      <ConfigurationHeader
        designator={
          revision && (
            <span
              className="font-mono text-xs font-semibold text-ds-text-primary"
              title={t('configuration:host.revisionTitle', { fingerprint: revision })}
            >
              {`REV ${revision.slice(0, 8)}`}
            </span>
          )
        }
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
        {recommendations.map((recommendation, index) => (
          <p key={index} className="text-xs text-ds-text-secondary">{recommendation}</p>
        ))}
      </ConfigurationHeader>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_20rem] xl:gap-6">
      {/* T9 change circuit: first in reading order so it sits above the
          form as a strip below xl; a sticky right column from xl up (from lg
          the rail already takes 16rem, so a column would squeeze the form). */}
      <ChangeCircuit
        className="xl:col-start-2 xl:row-start-1"
        circuit={circuit}
        pendingCount={diff.pendingCount}
        restartCount={restartCount}
        reviewDisabled={!diff.canSave || isSaving}
        onReview={handleReview}
        onDiscard={diff.pendingCount > 0 && !isSaving ? handleReset : undefined}
      />

      <div className="min-w-0 space-y-4 xl:col-start-1 xl:row-start-1">
      {/* Sections: a WAI-ARIA tablist with roving tabindex. */}
      <Tabs
        label={t('configuration:sectionsLabel')}
        idBase="config"
        items={SECTIONS}
        selected={activeTab}
        onSelect={setActiveTab}
      />

      <TabPanel idBase="config" selected={activeTab}>
        {readOnly ? (
          <div className="border-l-2 border-ds-warning py-1 pl-3 text-sm text-ds-on-tint-warning" role="status">
            {t('configuration:controlsUnavailable')}
          </div>
        ) : activeTab === 'ai' ? <AIProviderSettings /> : <ActiveComponent settings={formData} onUpdate={handleUpdateField} disabled={isSaving} configuredConsoleLevel={data.config?.runtimeConsoleLogging ? data.config.runtimeConsoleLogging.configuredLevel ?? undefined : data.config?.logging?.console?.level} runningConsoleLevel={data.config?.runtimeConsoleLogging?.runningLevel ?? undefined} runtimeConsoleLoggingStatus={data.config?.runtimeConsoleLogging?.status ?? 'unknown'} fieldErrors={fieldValidationMap(diff.fields)} />}
      </TabPanel>

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
            <h2 id={rawSettingsHeadingId} className="text-sm font-semibold text-ds-text-primary">
              {t('configuration:advanced.title')}
            </h2>
            <p className="text-xs font-semibold uppercase tracking-wide text-ds-text-primary">
              {t('configuration:advanced.directEdit')}
            </p>
            <p className="text-sm text-ds-text-secondary">
              {t('configuration:advanced.description')}
            </p>
          </div>
          {data.settingsDoc?.backupPath && (
            <span className="min-w-0 break-all text-xs text-ds-text-muted">
              {t('configuration:advanced.lastBackup', { path: data.settingsDoc.backupPath })}
            </span>
          )}
        </div>

        {/* The raw editor is an escape hatch: closed by default so it does
            not compete with the form under every tab. Its heading and the
            bypass statement above stay visible. */}
        <details className="group" data-testid="raw-settings-disclosure">
          <summary className="action-btn-secondary w-fit cursor-pointer list-none text-sm [&::-webkit-details-marker]:hidden">
            <ChevronDown className="h-4 w-4 transition-transform group-open:rotate-180" aria-hidden="true" />
            {t('configuration:advanced.disclosure')}
          </summary>
          <div className="mt-4 space-y-4">
            {/* Locked is the safe default, so the banner is neutral ink with the
                unlock control beside its text. A leading control-border rule,
                not a box: it sits flat on the sheet (DESIGN.md Lock banner). */}
            {!rawEditorUnlocked && (
              <div
                data-testid="raw-settings-locked-banner"
                className="flex flex-wrap items-center gap-x-3 gap-y-2 border-l-2 border-[var(--ds-border-strong)] py-1 pl-3 text-sm text-ds-text-secondary"
              >
                <Lock className="h-4 w-4 shrink-0 text-ds-text-primary" aria-hidden="true" />
                <span className="min-w-0 flex-[1_1_14rem]">{t('configuration:advanced.lockedBadge')}</span>
                <button
                  type="button"
                  onClick={() => setUnlockDialogOpen(true)}
                  disabled={!data.settingsDoc?.writable || actions.saveRawSettingsMutation.isPending}
                  data-testid="raw-settings-unlock-btn"
                  className="action-btn-secondary min-h-9 text-sm"
                >
                  <LockOpen className="h-4 w-4" aria-hidden="true" />
                  {t('configuration:advanced.unlockToEdit')}
                </button>
              </div>
            )}

            <textarea
              value={rawSettingsContent}
              onChange={(event) => {
                if (!rawEditorUnlocked) return;
                setRawSettingsContent(event.target.value);
                setRawParseError(null);
                if (actions.saveRawSettingsMutation.error) actions.saveRawSettingsMutation.reset();
              }}
              readOnly={!rawEditorUnlocked}
              aria-readonly={!rawEditorUnlocked}
              aria-labelledby={rawSettingsHeadingId}
              aria-invalid={rawError ? true : undefined}
              aria-describedby={rawError ? rawErrorId : undefined}
              // Sized to the source: one row per line between 8 and 24, then scroll.
              rows={Math.min(Math.max(rawSettingsContent.split('\n').length, 8), 24)}
              className={cn(
                'w-full resize-y overflow-auto rounded-box border border-border p-3 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-primary',
                rawEditorUnlocked ? 'bg-base-100 text-base-content' : 'cursor-not-allowed bg-ds-bg-base text-ds-text-secondary'
              )}
              spellCheck={false}
            />
            {rawError && (
              <p id={rawErrorId} role="alert" className="text-sm text-ds-on-tint-error" data-testid="raw-settings-error">
                {rawError}
              </p>
            )}

            {rawEditorUnlocked && (
              <div className="flex justify-end gap-2">
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
                  {actions.saveRawSettingsMutation.isPending ? t('configuration:advanced.savingRaw') : t('configuration:advanced.saveRaw')}
                </button>
              </div>
            )}
          </div>
        </details>
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
    </div>
    </LoadingBoundary>
  );
}
