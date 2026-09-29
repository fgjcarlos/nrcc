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
  authEnabled: 'Auth Enabled',
  authAdminUser: 'Auth Admin User',
  authAdminPassword: 'Auth Admin Password',
  authNodeHttpEnabled: 'Node HTTP Auth Enabled',
  authNodeHttpUser: 'Node HTTP Auth User',
  authNodeHttpPassword: 'Node HTTP Auth Password',
  authStaticEnabled: 'Static Auth Enabled',
  authStaticUser: 'Static Auth User',
  authStaticPassword: 'Static Auth Password',
  // Projects
  projectsEnabled: 'Projects Enabled',
  // Logging
  loggingConsoleLevel: 'Console Log Level',
  loggingConsoleMetrics: 'Console Metrics',
  loggingInternalLevel: 'Internal Log Level',
  loggingInternalMetrics: 'Internal Metrics',
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
