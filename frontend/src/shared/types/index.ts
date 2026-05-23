// ============================================
// API Response Types
// ============================================

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
  timestamp: string;
}

// ============================================
// LOGGING
// ============================================

export type LoggingLevel = 'trace' | 'debug' | 'info' | 'warn' | 'error' | 'fatal';

export interface LoggingHandler {
  level?: LoggingLevel;
  metrics?: boolean;
}

export interface LoggingSettings {
  loggingLevel?: LoggingLevel;
  console?: LoggingHandler;
  internal?: LoggingHandler;
}

// ============================================
// EDITOR THEME - Page
// ============================================

export interface EditorPage {
  title?: string;
  favicon?: string;
  css?: string | string[];
  scripts?: string | string[];
}

// ============================================
// EDITOR THEME - Header
// ============================================

export interface EditorHeader {
  title?: string;
  image?: string;
  url?: string;
}

// ============================================
// EDITOR THEME - Deploy Button
// ============================================

export interface EditorDeployButton {
  type?: 'default' | 'simple' | 'icon';
  label?: string;
  icon?: string;
}

// ============================================
// EDITOR THEME - Menu
// ============================================

export interface EditorMenuItem {
  label?: string;
  url?: string;
}

export interface EditorMenu {
  'menu-item-import-library'?: boolean | EditorMenuItem;
  'menu-item-export-library'?: boolean | EditorMenuItem;
  'menu-item-keyboard-shortcuts'?: boolean;
  'menu-item-help'?: boolean | EditorMenuItem;
  'menu-item-welcome'?: boolean;
  'menu-item-nodes'?: boolean;
  'menu-item-view'?: boolean;
  'menu-item-users'?: boolean;
  'menu-item-settings'?: boolean;
  'menu-item-install'?: boolean;
  'menu-item-project'?: boolean;
  'menu-item-subflow'?: boolean;
  'menu-item-examples'?: boolean;
}

// ============================================
// EDITOR THEME - Palette
// ============================================

export interface EditorPaletteCatalogue {
  id?: string;
  url: string;
  label?: string;
}

export interface EditorPaletteThemeItem {
  category?: string;
  type?: string;
  color?: string;
}

export interface EditorPalette {
  editable?: boolean;
  catalogues?: string[];
  theme?: EditorPaletteThemeItem[];
}

// ============================================
// EDITOR THEME - Projects
// ============================================

export interface EditorProjects {
  enabled?: boolean;
}

// ============================================
// EDITOR THEME - Code Editor
// ============================================

export interface EditorCodeEditorOptions {
  theme?: string;
  fontSize?: number;
  fontFamily?: string;
  tabSize?: number;
  minimap?: boolean;
  lineNumbers?: boolean;
  foldGutter?: boolean;
  wordWrap?: 'off' | 'on' | 'word' | 'char';
}

export interface EditorCodeEditor {
  lib?: 'ace' | 'monaco';
  options?: EditorCodeEditorOptions;
}

// ============================================
// EDITOR THEME - Login/Logout
// ============================================

export interface EditorLogin {
  image?: string;
}

export interface EditorLogout {
  redirect?: string;
}

// ============================================
// EDITOR THEME - Mermaid
// ============================================

export interface EditorMermaid {
  theme?: 'default' | 'base' | 'forest' | 'dark' | 'neutral';
}

// ============================================
// EDITOR THEME - Complete
// ============================================

export interface EditorTheme {
  page?: EditorPage;
  header?: EditorHeader;
  deployButton?: EditorDeployButton;
  menu?: EditorMenu;
  palette?: EditorPalette;
  projects?: EditorProjects;
  codeEditor?: EditorCodeEditor;
  theme?: string;
  userMenu?: boolean;
  tours?: boolean;
  login?: EditorLogin;
  logout?: EditorLogout;
  mermaid?: EditorMermaid;
}

// ============================================
// RUNTIME STATE
// ============================================

export interface RuntimeStateSettings {
  enabled?: boolean;
  file?: string;
}

// ============================================
// AUTHENTICATION
// ============================================

export interface AdminAuthSettings {
  user: string;
  password: string;
  tokens?: string[];
}

export interface NodeHttpAuthSettings {
  user: string;
  password: string;
}

export interface StaticAuthSettings {
  user: string;
  password: string;
}

export interface AuthSettings {
  adminAuth?: AdminAuthSettings;
  nodeHttpAuth?: NodeHttpAuthSettings | boolean;
  staticAuth?: StaticAuthSettings | boolean;
}

// ============================================
// NODE-RED CONFIG - Complete
// ============================================

export interface NodeRedConfig {
  uiPort: number;
  uiHost?: string;
  httpAdminRoot?: string | false;
  httpNodeRoot?: string | false;
  httpRoot?: string;
  disableEditor?: boolean;
  projectsEnabled: boolean;
  logging?: LoggingSettings;
  loggingLevel?: LoggingLevel;
  flowFile?: string;
  userDir?: string;
  nodesDir?: string;
  editorTheme?: EditorTheme;
  runtimeState?: RuntimeStateSettings;
  lang?: string;
  adminAuth?: AdminAuthSettings;
  nodeHttpAuth?: NodeHttpAuthSettings | boolean;
  staticAuth?: StaticAuthSettings | boolean;
  settingsPath?: string;
  settingsSource?: string;
}

// ============================================
// DEFAULT CONFIG
// ============================================

export const DEFAULT_CONFIG: NodeRedConfig = {
  uiPort: 1880,
  uiHost: '0.0.0.0',
  httpAdminRoot: '/',
  httpNodeRoot: '/',
  disableEditor: false,
  projectsEnabled: false,
  logging: {
    console: { level: 'info', metrics: false },
    internal: { level: 'info', metrics: false },
  },
  flowFile: 'flows.json',
};

// ============================================
// FORM DATA (for editing)
// ============================================

export interface NodeRedConfigFormData {
  uiPort: number;
  uiHost: string;
  httpAdminRoot: string;
  httpNodeRoot: string;
  disableEditor: boolean;
  projectsEnabled: boolean;
  loggingConsoleLevel: LoggingLevel;
  loggingConsoleMetrics: boolean;
  loggingInternalLevel: LoggingLevel;
  loggingInternalMetrics: boolean;
  flowFile: string;
  userDir: string;
  nodesDir: string;
  editorPageTitle: string;
  editorPageFavicon: string;
  editorPageCss: string;
  editorHeaderTitle: string;
  editorHeaderImage: string;
  editorHeaderUrl: string;
  editorDeployType: 'default' | 'simple' | 'icon';
  editorDeployLabel: string;
  editorDeployIcon: string;
  editorPaletteEditable: boolean;
  editorPaletteCatalogues: string;
  editorProjectsEnabled: boolean;
  editorCodeLib: 'ace' | 'monaco';
  editorCodeTheme: string;
  editorCodeFontSize: number;
  editorUserMenu: boolean;
  editorTours: boolean;
  editorLoginImage: string;
  editorLogoutRedirect: string;
  runtimeStateEnabled: boolean;
  runtimeStateFile: string;
  lang: string;
  authEnabled: boolean;
  authAdminUser: string;
  authAdminPassword: string;
  authNodeHttpEnabled: boolean;
  authNodeHttpUser: string;
  authNodeHttpPassword: string;
  authStaticEnabled: boolean;
  authStaticUser: string;
  authStaticPassword: string;
}

export function configToFormData(config: NodeRedConfig): NodeRedConfigFormData {
  return {
    uiPort: config.uiPort,
    uiHost: config.uiHost || '0.0.0.0',
    httpAdminRoot: typeof config.httpAdminRoot === 'string' ? config.httpAdminRoot : '/',
    httpNodeRoot: typeof config.httpNodeRoot === 'string' ? config.httpNodeRoot : '/',
    disableEditor: config.disableEditor || false,
    projectsEnabled: config.projectsEnabled,
    loggingConsoleLevel: config.logging?.console?.level || config.loggingLevel || 'info',
    loggingConsoleMetrics: config.logging?.console?.metrics || false,
    loggingInternalLevel: config.logging?.internal?.level || 'info',
    loggingInternalMetrics: config.logging?.internal?.metrics || false,
    flowFile: config.flowFile || 'flows.json',
    userDir: config.userDir || '',
    nodesDir: config.nodesDir || '',
    editorPageTitle: config.editorTheme?.page?.title || 'Node-RED',
    editorPageFavicon: config.editorTheme?.page?.favicon || '',
    editorPageCss: Array.isArray(config.editorTheme?.page?.css)
      ? config.editorTheme.page.css.join('\n')
      : config.editorTheme?.page?.css || '',
    editorHeaderTitle: config.editorTheme?.header?.title || 'Node-RED',
    editorHeaderImage: config.editorTheme?.header?.image || '',
    editorHeaderUrl: config.editorTheme?.header?.url || '',
    editorDeployType: config.editorTheme?.deployButton?.type || 'default',
    editorDeployLabel: config.editorTheme?.deployButton?.label || 'Deploy',
    editorDeployIcon: config.editorTheme?.deployButton?.icon || '',
    editorPaletteEditable: config.editorTheme?.palette?.editable !== false,
    editorPaletteCatalogues: config.editorTheme?.palette?.catalogues?.join('\n') || '',
    editorProjectsEnabled: config.editorTheme?.projects?.enabled || false,
    editorCodeLib: config.editorTheme?.codeEditor?.lib || 'ace',
    editorCodeTheme: config.editorTheme?.codeEditor?.options?.theme || 'vs',
    editorCodeFontSize: config.editorTheme?.codeEditor?.options?.fontSize || 12,
    editorUserMenu: config.editorTheme?.userMenu !== false,
    editorTours: config.editorTheme?.tours !== false,
    editorLoginImage: config.editorTheme?.login?.image || '',
    editorLogoutRedirect: config.editorTheme?.logout?.redirect || '',
    runtimeStateEnabled: config.runtimeState?.enabled || false,
    runtimeStateFile: config.runtimeState?.file || '',
    lang: config.lang || 'en-US',
    authEnabled: !!config.adminAuth,
    authAdminUser: config.adminAuth?.user || '',
    authAdminPassword: '',
    authNodeHttpEnabled: !!config.nodeHttpAuth && typeof config.nodeHttpAuth === 'object',
    authNodeHttpUser: typeof config.nodeHttpAuth === 'object' ? config.nodeHttpAuth?.user || '' : '',
    authNodeHttpPassword: '',
    authStaticEnabled: !!config.staticAuth && typeof config.staticAuth === 'object',
    authStaticUser: typeof config.staticAuth === 'object' ? config.staticAuth?.user || '' : '',
    authStaticPassword: '',
  };
}

export function formDataToConfig(formData: NodeRedConfigFormData): Partial<NodeRedConfig> {
  const config: Partial<NodeRedConfig> = {
    uiPort: formData.uiPort,
    uiHost: formData.uiHost || undefined,
    httpAdminRoot: formData.httpAdminRoot || '/',
    httpNodeRoot: formData.httpNodeRoot || '/',
    disableEditor: formData.disableEditor || undefined,
    projectsEnabled: formData.projectsEnabled,
    logging: {
      console: {
        level: formData.loggingConsoleLevel,
        metrics: formData.loggingConsoleMetrics || undefined,
      },
      internal: {
        level: formData.loggingInternalLevel,
        metrics: formData.loggingInternalMetrics || undefined,
      },
    },
    flowFile: formData.flowFile || undefined,
    userDir: formData.userDir || undefined,
    nodesDir: formData.nodesDir || undefined,
    editorTheme: {
      page: formData.editorPageTitle || formData.editorPageFavicon || formData.editorPageCss ? {
        title: formData.editorPageTitle || undefined,
        favicon: formData.editorPageFavicon || undefined,
        css: formData.editorPageCss || undefined,
      } : undefined,
      header: formData.editorHeaderTitle || formData.editorHeaderImage || formData.editorHeaderUrl ? {
        title: formData.editorHeaderTitle || undefined,
        image: formData.editorHeaderImage || undefined,
        url: formData.editorHeaderUrl || undefined,
      } : undefined,
      deployButton: formData.editorDeployType !== 'default' || formData.editorDeployLabel !== 'Deploy' ? {
        type: formData.editorDeployType,
        label: formData.editorDeployLabel || undefined,
        icon: formData.editorDeployIcon || undefined,
      } : undefined,
      palette: formData.editorPaletteCatalogues ? {
        editable: formData.editorPaletteEditable,
        catalogues: formData.editorPaletteCatalogues.split('\n').filter(Boolean),
      } : undefined,
      projects: formData.editorProjectsEnabled ? {
        enabled: true,
      } : undefined,
      codeEditor: formData.editorCodeLib !== 'ace' || formData.editorCodeTheme !== 'vs' ? {
        lib: formData.editorCodeLib,
        options: {
          theme: formData.editorCodeTheme,
          fontSize: formData.editorCodeFontSize,
        },
      } : undefined,
      userMenu: formData.editorUserMenu ? true : false,
      tours: formData.editorTours ? true : false,
      login: formData.editorLoginImage ? {
        image: formData.editorLoginImage,
      } : undefined,
      logout: formData.editorLogoutRedirect ? {
        redirect: formData.editorLogoutRedirect,
      } : undefined,
    },
    runtimeState: formData.runtimeStateEnabled ? {
      enabled: true,
      file: formData.runtimeStateFile || undefined,
    } : undefined,
    lang: formData.lang || undefined,
  };
  
  if (config.editorTheme) {
    Object.keys(config.editorTheme).forEach(key => {
      if (config.editorTheme && config.editorTheme[key as keyof typeof config.editorTheme] === undefined) {
        delete config.editorTheme[key as keyof typeof config.editorTheme];
      }
    });
    if (Object.keys(config.editorTheme).length === 0) {
      delete config.editorTheme;
    }
  }
  
  return config;
}

// ============================================
// RUNTIME TYPES
// ============================================

export type RuntimeStatus = 'running' | 'stopped' | 'error' | 'unknown' | 'detected';

export interface RuntimeInfo {
  status: RuntimeStatus;
  pid?: number;
  uptime: number;
  memory?: ProcessMemory;
  version?: string;
  installationMode?: InstallationMode;
  managedByNrcc?: boolean;
  detected?: boolean;
}

export interface ProcessMemory {
  rss: number;
  heapTotal: number;
  heapUsed: number;
  external: number;
}

// ============================================
// DOCKER TYPES
// ============================================

export type ContainerStatus = 'running' | 'exited' | 'paused' | 'created' | 'restarting' | 'removing' | 'dead';

export interface PortMapping {
  privatePort: number;
  publicPort?: number;
  type: 'tcp' | 'udp';
}

export interface ContainerInfo {
  id: string;
  name: string;
  image: string;
  status: ContainerStatus;
  created: string;
  ports: PortMapping[];
  state: ContainerState;
}

export interface ContainerState {
  running: boolean;
  paused: boolean;
  restartCount: number;
  memory: number;
  cpu: number;
}

// ============================================
// BOOTSTRAP / HOST DETECTION
// ============================================

export type InstallationMode = 'none' | 'native' | 'docker' | 'unknown';

export interface DependencyStatus {
  name: string;
  installed: boolean;
  version?: string;
  command?: string;
  details?: string;
}

export interface NodeRedEnvironment {
  detected: boolean;
  mode: InstallationMode;
  managedByNrcc: boolean;
  running: boolean;
  version?: string;
  executable?: string;
  containerName?: string;
  containerId?: string;
  userDir?: string;
  settingsPath?: string;
}

export interface SettingsDocument {
  path: string;
  source: string;
  writable: boolean;
  backupPath?: string;
  content?: string;
}

export interface HostStatus {
  platform: string;
  ready: boolean;
  interactive: boolean;
  nodejs: DependencyStatus;
  npm: DependencyStatus;
  nodeRedBinary: DependencyStatus;
  docker: DependencyStatus;
  dockerCompose: DependencyStatus;
  nodeRed: NodeRedEnvironment;
  settings: SettingsDocument;
  recommendations?: string[];
}

export interface DockerInfo {
  version: string;
  containersRunning: number;
  containersPaused: number;
  containersStopped: number;
  images: number;
}

// ============================================
// SYSTEM TYPES
// ============================================

export interface SystemInfo {
  cpu: {
    usage: number;
    cores: number;
  };
  memory: {
    total: number;
    used: number;
    free: number;
    usagePercent: number;
  };
  disk: {
    total: number;
    used: number;
    free: number;
    usagePercent: number;
  };
  uptime: number;
  platform: string;
  hostname: string;
}

// ============================================
// LOG TYPES
// ============================================

export type LogLevel = 'debug' | 'info' | 'warn' | 'error';

export interface LogEntry {
  id: string;
  timestamp: string;
  level: LogLevel;
  message: string;
  source?: string;
}
