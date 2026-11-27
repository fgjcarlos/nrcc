import type { NodeRedConfigFormData } from '@/shared/types';

/**
 * Issue #766 slice F (W4) — readable labels for every form field.
 *
 * The `<ReviewChangesPanel>` renders fields grouped by restart
 * severity without knowing which tab they belong to, so it needs a
 * flat label map keyed by `keyof NodeRedConfigFormData`. The values
 * are intentionally plain English — these are debug-facing identifiers
 * the operator has seen for the whole slice A–F arc, and i18n for the
 * field taxonomy itself is tracked separately (see issue #767).
 *
 * Keep this list aligned with the `label="..."` literals in
 * `BasicSettings.tsx` and `SecuritySettings.tsx`.
 */
export const FIELD_LABELS: Record<keyof NodeRedConfigFormData, string> = {
  // Basic
  uiPort: 'UI Port',
  uiHost: 'UI Host',
  httpAdminRoot: 'Admin Root',
  httpNodeRoot: 'Node Root',
  disableEditor: 'Disable Editor',
  // Auth
  // Projects
  projectsEnabled: 'Projects Enabled',
  // Logging
  loggingConsoleLevel: 'Console Log Level',
  loggingConsoleMetrics: 'Console Metrics',
  loggingConsoleAudit: 'Console Audit',
  loggingInternalLevel: 'Internal Log Level',
  loggingInternalMetrics: 'Internal Metrics',
  loggingInternalAudit: 'Internal Audit',
  // Files
  flowFile: 'Flow File',
  userDir: 'User Dir',
  nodesDir: 'Nodes Dir',
  // Editor theme (runtime-reload)
  editorPageTitle: 'Editor Page Title',
  editorPageFavicon: 'Editor Page Favicon',
  editorPageCss: 'Editor Page CSS',
  editorHeaderTitle: 'Editor Header Title',
  editorHeaderImage: 'Editor Header Image',
  editorHeaderUrl: 'Editor Header URL',
  editorDeployType: 'Editor Deploy Type',
  editorDeployLabel: 'Editor Deploy Label',
  editorDeployIcon: 'Editor Deploy Icon',
  editorPaletteEditable: 'Palette Editable',
  editorPaletteCatalogues: 'Palette Catalogues',
  editorProjectsEnabled: 'Editor Projects Enabled',
  editorCodeLib: 'Editor Code Library',
  editorCodeTheme: 'Editor Code Theme',
  editorCodeFontSize: 'Editor Code Font Size',
  editorUserMenu: 'Editor User Menu',
  editorTours: 'Editor Tours',
  editorLoginImage: 'Editor Login Image',
  editorLogoutRedirect: 'Editor Logout Redirect',
  // Runtime state
  runtimeStateEnabled: 'Runtime State Enabled',
  runtimeStateFile: 'Runtime State File',
  // Language
  lang: 'Language',
  // Security (issue #762)
  credentialSecret: 'Credential Secret',
  requireHttps: 'Require HTTPS',
  httpsKey: 'Private Key Path',
  httpsCert: 'Certificate Path',
  httpsCA: 'CA Bundle Path',
  httpsPort: 'HTTPS Port',
  httpsPassphrase: 'Private Key Passphrase',
};

/**
 * T9 finish review — the settings.js key each form field writes, shown as
 * the field's terminal label (IEC designator). Paths follow
 * `formDataToConfigPayload` and the backend renderer
 * (internal/service/config.go renderSettingsJS, renderHttpsBlock).
 */
export const SETTINGS_KEYS: Record<keyof NodeRedConfigFormData, string> = {
  uiPort: 'uiPort',
  uiHost: 'uiHost',
  httpAdminRoot: 'httpAdminRoot',
  httpNodeRoot: 'httpNodeRoot',
  disableEditor: 'disableEditor',
  projectsEnabled: 'projectsEnabled',
  loggingConsoleLevel: 'logging.console.level',
  loggingConsoleMetrics: 'logging.console.metrics',
  loggingConsoleAudit: 'logging.console.audit',
  loggingInternalLevel: 'logging.internal.level',
  loggingInternalMetrics: 'logging.internal.metrics',
  loggingInternalAudit: 'logging.internal.audit',
  flowFile: 'flowFile',
  userDir: 'userDir',
  nodesDir: 'nodesDir',
  editorPageTitle: 'editorTheme.page.title',
  editorPageFavicon: 'editorTheme.page.favicon',
  editorPageCss: 'editorTheme.page.css',
  editorHeaderTitle: 'editorTheme.header.title',
  editorHeaderImage: 'editorTheme.header.image',
  editorHeaderUrl: 'editorTheme.header.url',
  editorDeployType: 'editorTheme.deployButton.type',
  editorDeployLabel: 'editorTheme.deployButton.label',
  editorDeployIcon: 'editorTheme.deployButton.icon',
  editorPaletteEditable: 'editorTheme.palette.editable',
  editorPaletteCatalogues: 'editorTheme.palette.catalogues',
  editorProjectsEnabled: 'editorTheme.projects.enabled',
  editorCodeLib: 'editorTheme.codeEditor.lib',
  editorCodeTheme: 'editorTheme.codeEditor.options.theme',
  editorCodeFontSize: 'editorTheme.codeEditor.options.fontSize',
  editorUserMenu: 'editorTheme.userMenu',
  editorTours: 'editorTheme.tours',
  editorLoginImage: 'editorTheme.login.image',
  editorLogoutRedirect: 'editorTheme.logout.redirect',
  runtimeStateEnabled: 'runtimeState.enabled',
  runtimeStateFile: 'runtimeState.file',
  lang: 'lang',
  credentialSecret: 'credentialSecret',
  requireHttps: 'requireHttps',
  httpsKey: 'https.key',
  httpsCert: 'https.cert',
  httpsCA: 'https.ca',
  httpsPort: 'https.port',
  httpsPassphrase: 'https.passphrase',
};
