import type { NodeRedConfigFormData, LoggingLevel, EditorTheme, RuntimeStateSettings } from '@/shared/types';

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

  // Page settings (title, favicon)
  const page: Record<string, unknown> = {};
  if (formData.editorPageTitle) page.title = formData.editorPageTitle;
  if (formData.editorPageFavicon) page.favicon = formData.editorPageFavicon;
  if (formData.editorPageCss) page.css = formData.editorPageCss;
  if (Object.keys(page).length > 0) editorTheme.page = page;

  // Header settings (title, image, url)
  const header: Record<string, unknown> = {};
  if (formData.editorHeaderTitle) header.title = formData.editorHeaderTitle;
  if (formData.editorHeaderImage) header.image = formData.editorHeaderImage;
  if (formData.editorHeaderUrl) header.url = formData.editorHeaderUrl;
  if (Object.keys(header).length > 0) editorTheme.header = header;

  // Login/Logout settings
  if (formData.editorLoginImage) {
    editorTheme.login = { image: formData.editorLoginImage };
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
  if (formData.editorCodeLib !== 'ace') {
    editorTheme.codeEditor = {
      lib: formData.editorCodeLib,
      options: { theme: formData.editorCodeTheme },
    };
  }
  if (!formData.editorUserMenu) editorTheme.userMenu = false;
  if (!formData.editorTours) editorTheme.tours = false;

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
export function configToFormData(config: Record<string, unknown>): NodeRedConfigFormData {
  return {
    uiPort: config.uiPort as number,
    uiHost: config.uiHost as string || '0.0.0.0',
    httpAdminRoot: typeof config.httpAdminRoot === 'string' ? config.httpAdminRoot : '/',
    httpNodeRoot: typeof config.httpNodeRoot === 'string' ? config.httpNodeRoot : '/',
    disableEditor: (config.disableEditor as boolean) || false,

    // Authentication — password nunca se precarga (no exponemos hashes)
    authEnabled: !!(config.adminAuth as any),
    authAdminUser: (config.adminAuth as any)?.users?.[0]?.username || '',
    authAdminPassword: '', // vacío: si no cambia, el backend preserva el hash
    authNodeHttpEnabled: !!(config.nodeHttpAuth as any),
    authNodeHttpUser: (config.nodeHttpAuth as any)?.users?.[0]?.username || '',
    authNodeHttpPassword: '',
    authStaticEnabled: !!(config.staticAuth as any),
    authStaticUser: (config.staticAuth as any)?.users?.[0]?.username || '',
    authStaticPassword: '',

    projectsEnabled: (config.projectsEnabled as boolean) || false,

    loggingConsoleLevel: (config.logging as any)?.console?.level || 'info',
    loggingConsoleMetrics: (config.logging as any)?.console?.metrics || false,
    loggingInternalLevel: (config.logging as any)?.internal?.level || 'info',
    loggingInternalMetrics: (config.logging as any)?.internal?.metrics || false,

    flowFile: (config.flowFile as string) || 'flows.json',
    userDir: (config.userDir as string) || '',
    nodesDir: (config.nodesDir as string) || '',

    editorPageTitle: (config.editorTheme as any)?.page?.title || 'Node-RED',
    editorPageFavicon: (config.editorTheme as any)?.page?.favicon || '',
    editorPageCss: '',
    editorHeaderTitle: (config.editorTheme as any)?.header?.title || 'Node-RED',
    editorHeaderImage: (config.editorTheme as any)?.header?.image || '',
    editorHeaderUrl: (config.editorTheme as any)?.header?.url || '',
    editorDeployType: (config.editorTheme as any)?.deployButton?.type || 'default',
    editorDeployLabel: (config.editorTheme as any)?.deployButton?.label || 'Deploy',
    editorDeployIcon: (config.editorTheme as any)?.deployButton?.icon || '',
    editorPaletteEditable: (config.editorTheme as any)?.palette?.editable !== false,
    editorPaletteCatalogues: ((config.editorTheme as any)?.palette?.catalogues as string[] || []).join('\n'),
    editorProjectsEnabled: (config.editorTheme as any)?.projects?.enabled || false,
    editorCodeLib: (config.editorTheme as any)?.codeEditor?.lib || 'ace',
    editorCodeTheme: (config.editorTheme as any)?.codeEditor?.options?.theme || 'vs',
    editorCodeFontSize: (config.editorTheme as any)?.codeEditor?.options?.fontSize || 12,
    editorUserMenu: (config.editorTheme as any)?.userMenu !== false,
    editorTours: (config.editorTheme as any)?.tours !== false,
    editorLoginImage: (config.editorTheme as any)?.login?.image || '',
    editorLogoutRedirect: (config.editorTheme as any)?.logout?.redirect || '',

    runtimeStateEnabled: (config.runtimeState as any)?.enabled || false,
    runtimeStateFile: (config.runtimeState as any)?.file || '',
    lang: (config.lang as string) || 'en-US',
  };
}
