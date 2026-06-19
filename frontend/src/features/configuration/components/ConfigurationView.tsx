import { useState, useEffect } from 'react';
import { cn } from '@/shared/lib';
import type { NodeRedConfigFormData } from '@/shared/types';
import {
  BasicSettings,
  AuthSettings,
  LoggingSettings,
  EditorThemeSettings,
} from '.';
import {
  Settings, Server, Lock, Activity, Palette,
  Save, LockOpen, AlertTriangle
} from 'lucide-react';
import { useConfigurationData, useConfigurationActions } from '../hooks';
import { UI_COPY } from '@/shared/constants/uiCopy';
import { ConfirmationDialog } from '@/shared/components/ConfirmationDialog';

// ============================================
// Sections Configuration
// ============================================

interface Section {
  id: string;
  label: string;
  icon: React.ElementType;
  component: React.ComponentType<{
    settings: NodeRedConfigFormData;
    onUpdate: (field: keyof NodeRedConfigFormData, value: string | number | boolean) => void;
    disabled?: boolean;
  }>;
}

const SECTIONS: Section[] = [
  { id: 'basic', label: 'Basic', icon: Server, component: BasicSettings },
  { id: 'auth', label: 'Authentication', icon: Lock, component: AuthSettings },
  { id: 'logging', label: 'Logging', icon: Activity, component: LoggingSettings },
  { id: 'editor', label: 'Editor Theme', icon: Palette, component: EditorThemeSettings },
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
  authEnabled: false,
  authAdminUser: '',
  authAdminPassword: '',
  authNodeHttpEnabled: false,
  authNodeHttpUser: '',
  authNodeHttpPassword: '',
  authStaticEnabled: false,
  authStaticUser: '',
  authStaticPassword: '',

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
  const [hasChanges, setHasChanges] = useState(false);
  const [rawSettingsContent, setRawSettingsContent] = useState('');
  // Raw-settings editor gate (issue #364): the textarea is read-only by
  // default. The user must click "Unlock to edit settings.js", acknowledge
  // the warning dialog, and only then can they edit and save. Re-locking
  // discards any in-flight edits.
  const [rawEditorUnlocked, setRawEditorUnlocked] = useState(false);
  const [rawEditorSnapshot, setRawEditorSnapshot] = useState('');
  const [unlockDialogOpen, setUnlockDialogOpen] = useState(false);

  // Data and actions hooks
  const data = useConfigurationData();
  const actions = useConfigurationActions();

  // Sync loaded config to form
  useEffect(() => {
    if (data.initialFormData) {
      setFormData(data.initialFormData);
      setHasChanges(false);
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
    setHasChanges(true);
  };

  const handleSave = async () => {
    await actions.handleSave(formData);
    setHasChanges(false);
  };

  const handleReset = () => {
    setFormData(INITIAL_FORM_DATA);
    setHasChanges(false);
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
    actions.handleSaveRawSettings(content);
    // Snapshot the saved content so a re-lock restores the saved state,
    // not the pre-edit state.
    setRawEditorSnapshot(content);
  };

  // Derived state
  const ActiveComponent = SECTIONS.find(s => s.id === activeTab)?.component || BasicSettings;
  const isLoading = data.configLoading;
  const isSaving = actions.saveConfigMutation.isPending;

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-[0.24em] text-base-content/50">Settings</p>
          <h1 className="flex items-center gap-3 text-2xl font-bold text-base-content">
            <Settings className="h-6 w-6" />
            Node-RED Configuration
          </h1>
        </div>
        <div className="flex items-center gap-2">
          {hasChanges && (
            <button
              type="button"
              onClick={handleReset}
              className="px-4 py-2 text-sm font-medium text-base-content/60 transition-colors hover:text-base-content"
            >
              Discard Changes
            </button>
          )}
          <button
            type="button"
            onClick={handleSave}
            disabled={!hasChanges || isSaving}
            className="action-btn-primary"
          >
            <Save className="w-4 h-4" />
            {actions.saveConfigMutation.isPending ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>

      {/* Host Status Info */}
      {data.hostStatus && (
        <div className="rounded-2xl border border-border bg-base-200/35 p-4">
          <div className="flex flex-col gap-2 text-sm text-base-content/75">
            <span>
              {UI_COPY.installationDetected}: <strong className="text-base-content">{data.hostStatus.nodeRed.mode}</strong>
              {data.hostStatus.nodeRed.detected ? '' : ` ${UI_COPY.nodeRedNotDetected}`}
            </span>
            <span>
              settings.js: <strong className="text-base-content">{data.hostStatus.settings.path || UI_COPY.pathNotDetected}</strong>
            </span>
            {data.hostStatus.recommendations && data.hostStatus.recommendations.length > 0 && (
              <span>{data.hostStatus.recommendations[0]}</span>
            )}
          </div>
        </div>
      )}

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
      <div className="surface-card p-6">
        <ActiveComponent
          settings={formData}
          onUpdate={handleUpdateField}
          disabled={isSaving}
        />
      </div>

      {/* Raw Settings Editor — gated by issue #364 */}
      <div className="surface-card p-6 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-base-content">
              {UI_COPY.advancedSettingsTitle}
            </h2>
            <p className="text-sm text-base-content/65">
              {UI_COPY.advancedSettingsDescription}
            </p>
          </div>
          {data.settingsDoc?.backupPath && (
            <span className="text-xs text-base-content/55">
              {UI_COPY.lastBackup(data.settingsDoc.backupPath)}
            </span>
          )}
        </div>

        {!rawEditorUnlocked && (
          <div
            data-testid="raw-settings-locked-banner"
            className="flex items-center gap-2 rounded-xl border border-warning/40 bg-warning/10 px-3 py-2 text-sm text-warning"
          >
            <AlertTriangle className="h-4 w-4 flex-shrink-0" />
            <span>{UI_COPY.lockedBadge}</span>
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
              {UI_COPY.unlockToEdit}
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
                {UI_COPY.cancelEdit}
              </button>
              <button
                type="button"
                onClick={() => handleSaveRawSettingsLocked(rawSettingsContent)}
                disabled={!data.settingsDoc?.writable || actions.saveRawSettingsMutation.isPending}
                data-testid="raw-settings-save-btn"
                className="action-btn-primary"
              >
                <Save className="w-4 h-4" />
                {actions.saveRawSettingsMutation.isPending ? UI_COPY.savingRawSettings : UI_COPY.saveChanges}
              </button>
            </>
          )}
        </div>
      </div>

      <ConfirmationDialog
        isOpen={unlockDialogOpen}
        title={UI_COPY.unlockDialogTitle}
        description={UI_COPY.unlockDialogDescription(data.settingsDoc?.backupPath || '/var/backups/nrcc/settings.js')}
        acknowledgement={UI_COPY.unlockDialogAcknowledgement}
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
    </div>
  );
}
