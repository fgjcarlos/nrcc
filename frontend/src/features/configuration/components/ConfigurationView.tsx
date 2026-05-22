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
  Save, Globe
} from 'lucide-react';
import { useConfigurationData, useConfigurationActions } from '../hooks';

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

  const handleSaveRawSettings = (content: string) => {
    actions.handleSaveRawSettings(content);
  };

  // Derived state
  const ActiveComponent = SECTIONS.find(s => s.id === activeTab)?.component || BasicSettings;
  const isLoading = data.configLoading;
  const isSaving = actions.saveConfigMutation.isPending || actions.restartMutation.isPending;

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
              Instalación detectada: <strong className="text-base-content">{data.hostStatus.nodeRed.mode}</strong>
              {data.hostStatus.nodeRed.detected ? '' : ' (sin Node-RED detectado)'}
            </span>
            <span>
              settings.js: <strong className="text-base-content">{data.hostStatus.settings.path || 'sin ruta detectada'}</strong>
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

      {/* Raw Settings Editor */}
      <div className="surface-card p-6 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-base-content">Advanced `settings.js`</h2>
            <p className="text-sm text-base-content/65">
              Edita el archivo real detectado por nrcc. Se crea backup antes de guardar.
            </p>
          </div>
          {data.settingsDoc?.backupPath && (
            <span className="text-xs text-base-content/55">Último backup: {data.settingsDoc.backupPath}</span>
          )}
        </div>

        <textarea
          value={rawSettingsContent}
          onChange={(event) => setRawSettingsContent(event.target.value)}
          className="min-h-[20rem] w-full rounded-2xl border border-border bg-base-100/80 p-4 font-mono text-sm text-base-content focus:outline-none focus:ring-2 focus:ring-primary"
          spellCheck={false}
        />

        <div className="flex justify-end">
          <button
            type="button"
            onClick={() => handleSaveRawSettings(rawSettingsContent)}
            disabled={!data.settingsDoc?.writable || actions.saveRawSettingsMutation.isPending}
            className="action-btn-secondary"
          >
            <Globe className="w-4 h-4" />
            {actions.saveRawSettingsMutation.isPending ? 'Saving settings.js...' : 'Save raw settings.js'}
          </button>
        </div>
      </div>
    </div>
  );
}
