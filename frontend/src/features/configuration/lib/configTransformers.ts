import type { NodeRedConfigFormData, LoggingLevel, EditorTheme, RuntimeStateSettings } from '@/shared/types';

/**
 * Shape of the raw Node-RED config response consumed by configToFormData.
 *
 * This models the READ response, which is external data: enum-like fields
 * (log level, code editor lib, deploy type) arrive as plain strings and are
 * narrowed to the domain unions at the point of use. Auth is a credentials
 * object with a `users` array (distinct from the write-side AdminAuthSettings).
 * The index signature keeps the door open for the many other Node-RED settings
 * without resorting to `as any` on the fields we actually read.
 */
interface CredentialsAuthResponse {
  users?: Array<{ username?: string }>;
}

interface LoggingHandlerResponse {
  level?: string;
  metrics?: boolean;
}

interface EditorThemeResponse {
  page?: { title?: string; favicon?: string };
  header?: { title?: string; image?: string; url?: string };
  deployButton?: { type?: string; label?: string; icon?: string };
  palette?: { editable?: boolean; catalogues?: string[] };
  projects?: { enabled?: boolean };
  codeEditor?: { lib?: string; options?: { theme?: string; fontSize?: number } };
  userMenu?: boolean;
  tours?: boolean;
  login?: { image?: string };
  logout?: { redirect?: string };
}

export interface NodeRedConfigResponse {
  uiPort?: number;
  uiHost?: string;
  httpAdminRoot?: string | false;
  httpNodeRoot?: string | false;
  disableEditor?: boolean;
  projectsEnabled?: boolean;
  adminAuth?: CredentialsAuthResponse;
  nodeHttpAuth?: CredentialsAuthResponse;
  staticAuth?: CredentialsAuthResponse;
  logging?: { console?: LoggingHandlerResponse; internal?: LoggingHandlerResponse };
  flowFile?: string;
  userDir?: string;
  nodesDir?: string;
  editorTheme?: EditorThemeResponse;
  runtimeState?: RuntimeStateSettings;
  lang?: string;
  [key: string]: unknown;
}

/**
 * API payload shape for admin/node-http/static auth
 */
export interface AuthPayloadUser {
  username: string;
  password: string;
  permissions?: string;
}

export interface AuthPayload {
  type: string;
  users: AuthPayloadUser[];
}

/**
 * Typed payload returned by formDataToConfigPayload
 */
export interface ConfigPayload {
  uiPort: number;
  uiHost: string;
  httpAdminRoot: string;
  httpNodeRoot: string;
  disableEditor?: boolean;
  projectsEnabled: boolean;
  logging?: {
    console?: { level: LoggingLevel; metrics?: boolean };
    internal?: { level: LoggingLevel; metrics?: boolean };
  };
  flowFile: string;
  userDir?: string;
  nodesDir?: string;
  editorTheme?: EditorTheme;
  runtimeState?: RuntimeStateSettings;
  lang: string;
  adminAuth?: AuthPayload;
  nodeHttpAuth?: AuthPayload;
  staticAuth?: AuthPayload;
}

/**
 * Convert form data to API payload
 * Pure function with no React dependencies
 */
function toEditorImageUrl(value: string): string {
  if (!value) return value;
  // Strip the "/uploads/" prefix for editorTheme — those URLs must resolve
  // from Node-RED's origin (1880), which serves them via httpStatic.
  if (value.startsWith('/uploads/')) return value.slice('/uploads'.length);
  return value;
}

export function formDataToConfigPayload(formData: NodeRedConfigFormData): ConfigPayload {
  const config: Record<string, unknown> = {};

  // Basic
  config.uiPort = formData.uiPort;
  config.uiHost = formData.uiHost || '0.0.0.0';
  config.httpAdminRoot = formData.httpAdminRoot || '/';
  config.httpNodeRoot = formData.httpNodeRoot || '/';
  if (formData.disableEditor) config.disableEditor = true;

  // Authentication
  // Si authEnabled pero password vacío → mandamos solo el username para indicar
  // que la auth sigue activa pero SIN cambiar el password (backend lo preserva)
  if (formData.authEnabled && formData.authAdminUser) {
    config.adminAuth = {
      type: 'credentials',
      users: [{
        username: formData.authAdminUser,
        password: formData.authAdminPassword || '', // vacío = backend preserva hash
        permissions: '*'
      }]
    };
  }

  if (formData.authNodeHttpEnabled && formData.authNodeHttpUser && formData.authNodeHttpPassword) {
    config.nodeHttpAuth = {
      type: 'credentials',
      users: [{
        username: formData.authNodeHttpUser,
        password: formData.authNodeHttpPassword
      }]
    };
  }

  if (formData.authStaticEnabled && formData.authStaticUser && formData.authStaticPassword) {
    config.staticAuth = {
      type: 'credentials',
      users: [{
        username: formData.authStaticUser,
        password: formData.authStaticPassword
      }]
    };
  }

  // Projects
  config.projectsEnabled = formData.projectsEnabled;

  // Logging - Multiple handlers
  config.logging = {
    console: {
      level: formData.loggingConsoleLevel,
      metrics: formData.loggingConsoleMetrics || undefined,
    },
    internal: {
      level: formData.loggingInternalLevel,
      metrics: formData.loggingInternalMetrics || undefined,
    },
  };

  // Files
  config.flowFile = formData.flowFile || 'flows.json';
  if (formData.userDir) config.userDir = formData.userDir;
  if (formData.nodesDir) config.nodesDir = formData.nodesDir;

  // Editor Theme
  const editorTheme: Record<string, unknown> = {};

  // Page settings (title, favicon, css). Use `!== undefined` rather than
  // truthy so clearing an input to "" actually clears the persisted value
  // instead of silently keeping the previous one.
  const page: Record<string, unknown> = {};
  if (formData.editorPageTitle !== undefined) page.title = formData.editorPageTitle;
  if (formData.editorPageFavicon !== undefined) page.favicon = toEditorImageUrl(formData.editorPageFavicon);
  if (formData.editorPageCss !== undefined) page.css = formData.editorPageCss;
  if (Object.keys(page).length > 0) editorTheme.page = page;

  // Header settings (title, image, url). Same explicit-undefined rule so
  // emptying the Header Title field actually clears it server-side.
  const header: Record<string, unknown> = {};
  if (formData.editorHeaderTitle !== undefined) header.title = formData.editorHeaderTitle;
  if (formData.editorHeaderImage !== undefined) header.image = toEditorImageUrl(formData.editorHeaderImage);
  if (formData.editorHeaderUrl !== undefined) header.url = formData.editorHeaderUrl;
  if (Object.keys(header).length > 0) editorTheme.header = header;

  // Deploy Button (#707: previously dropped on the floor by the transformer).
  if (
    formData.editorDeployType !== undefined ||
    formData.editorDeployLabel !== undefined ||
    formData.editorDeployIcon !== undefined
  ) {
    const deployButton: Record<string, unknown> = {};
    if (formData.editorDeployType !== undefined) deployButton.type = formData.editorDeployType;
    if (formData.editorDeployLabel !== undefined) deployButton.label = formData.editorDeployLabel;
    if (formData.editorDeployIcon !== undefined) deployButton.icon = formData.editorDeployIcon;
    editorTheme.deployButton = deployButton;
  }

  // Login/Logout settings
  if (formData.editorLoginImage) {
    editorTheme.login = { image: toEditorImageUrl(formData.editorLoginImage) };
  }
  if (formData.editorLogoutRedirect) {
    editorTheme.logout = { redirect: formData.editorLogoutRedirect };
  }

  if (formData.editorPaletteCatalogues) {
    editorTheme.palette = {
      editable: formData.editorPaletteEditable,
      catalogues: formData.editorPaletteCatalogues.split('\n').filter(Boolean),
    };
  }
  if (formData.editorProjectsEnabled) {
    editorTheme.projects = { enabled: true };
  }
  // Code Editor: include fontSize when monaco and the user actually set it
  // (#707: previously missing). Also include userMenu/tours in BOTH
  // directions so toggling back to true actually persists (the old
  // `if (!formData.editorUserMenu)` elided the on value).
  if (formData.editorCodeLib !== 'ace') {
    const codeEditor: Record<string, unknown> = {
      lib: formData.editorCodeLib,
    };
    const codeOptions: Record<string, unknown> = {};
    if (formData.editorCodeTheme) codeOptions.theme = formData.editorCodeTheme;
    if (typeof formData.editorCodeFontSize === 'number' && formData.editorCodeFontSize > 0) {
      codeOptions.fontSize = formData.editorCodeFontSize;
    }
    codeEditor.options = codeOptions;
    editorTheme.codeEditor = codeEditor;
  } else {
    // Even with ace, still forward fontSize so users who switch libraries
    // later see their preferred size.
    if (typeof formData.editorCodeFontSize === 'number' && formData.editorCodeFontSize > 0) {
      editorTheme.codeEditor = { options: { fontSize: formData.editorCodeFontSize } };
    }
  }
  // Always forward the boolean state of userMenu / tours so toggling back
  // to true after a previous false persists. Closes #707 (was: only sent
  // when false).
  if (formData.editorUserMenu !== undefined) editorTheme.userMenu = formData.editorUserMenu;
  if (formData.editorTours !== undefined) editorTheme.tours = formData.editorTours;

  if (Object.keys(editorTheme).length > 0) {
    config.editorTheme = editorTheme;
  }

  // Runtime State
  if (formData.runtimeStateEnabled) {
    config.runtimeState = {
      enabled: true,
      file: formData.runtimeStateFile || undefined,
    };
  }

  // Language
  config.lang = formData.lang || 'en-US';

  return config as unknown as ConfigPayload;
}

/**
 * Convert API config response to form data
 * Pure function with no React dependencies
 */
export function configToFormData(config: NodeRedConfigResponse): NodeRedConfigFormData {
  const editor = config.editorTheme;
  return {
    uiPort: config.uiPort ?? 1880,
    uiHost: config.uiHost || '0.0.0.0',
    httpAdminRoot: typeof config.httpAdminRoot === 'string' ? config.httpAdminRoot : '/',
    httpNodeRoot: typeof config.httpNodeRoot === 'string' ? config.httpNodeRoot : '/',
    disableEditor: config.disableEditor || false,

    // Authentication — password nunca se precarga (no exponemos hashes)
    authEnabled: !!config.adminAuth,
    authAdminUser: config.adminAuth?.users?.[0]?.username || '',
    authAdminPassword: '', // vacío: si no cambia, el backend preserva el hash
    authNodeHttpEnabled: !!config.nodeHttpAuth,
    authNodeHttpUser: config.nodeHttpAuth?.users?.[0]?.username || '',
    authNodeHttpPassword: '',
    authStaticEnabled: !!config.staticAuth,
    authStaticUser: config.staticAuth?.users?.[0]?.username || '',
    authStaticPassword: '',

    projectsEnabled: config.projectsEnabled || false,

    loggingConsoleLevel: (config.logging?.console?.level as LoggingLevel) || 'info',
    loggingConsoleMetrics: config.logging?.console?.metrics || false,
    loggingInternalLevel: (config.logging?.internal?.level as LoggingLevel) || 'info',
    loggingInternalMetrics: config.logging?.internal?.metrics || false,

    flowFile: config.flowFile || 'flows.json',
    userDir: config.userDir || '',
    nodesDir: config.nodesDir || '',

    editorPageTitle: editor?.page?.title || 'Node-RED',
    editorPageFavicon: editor?.page?.favicon || '',
    editorPageCss: '',
    editorHeaderTitle: editor?.header?.title || 'Node-RED',
    editorHeaderImage: editor?.header?.image || '',
    editorHeaderUrl: editor?.header?.url || '',
    editorDeployType: (editor?.deployButton?.type as NodeRedConfigFormData['editorDeployType']) || 'default',
    editorDeployLabel: editor?.deployButton?.label || 'Deploy',
    editorDeployIcon: editor?.deployButton?.icon || '',
    editorPaletteEditable: editor?.palette?.editable !== false,
    editorPaletteCatalogues: (editor?.palette?.catalogues || []).join('\n'),
    editorProjectsEnabled: editor?.projects?.enabled || false,
    editorCodeLib: (editor?.codeEditor?.lib as NodeRedConfigFormData['editorCodeLib']) || 'ace',
    editorCodeTheme: editor?.codeEditor?.options?.theme || 'vs',
    editorCodeFontSize: editor?.codeEditor?.options?.fontSize || 12,
    editorUserMenu: editor?.userMenu !== false,
    editorTours: editor?.tours !== false,
    editorLoginImage: editor?.login?.image || '',
    editorLogoutRedirect: editor?.logout?.redirect || '',

    runtimeStateEnabled: config.runtimeState?.enabled || false,
    runtimeStateFile: config.runtimeState?.file || '',
    lang: config.lang || 'en-US',
  };
}
