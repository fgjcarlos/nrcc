import { describe, it, expect } from 'vitest';
import { formDataToConfigPayload, configToFormData } from './configTransformers';
import type { NodeRedConfigFormData } from '@/shared/types';

/**
 * Test suite for configuration transformers
 * These are pure functions that convert between API payloads and form data
 */

describe('configTransformers', () => {
  // ============================================
  // configToFormData Tests
  // ============================================

  describe('configToFormData', () => {
    it('should convert a complete config to form data with all fields populated', () => {
      const completeConfig = {
        uiPort: 1880,
        uiHost: '192.168.1.100',
        httpAdminRoot: '/admin',
        httpNodeRoot: '/api',
        disableEditor: false,
        projectsEnabled: true,
        logging: {
          console: {
            level: 'debug',
            metrics: true,
          },
          internal: {
            level: 'info',
            metrics: true, // true so it's not turned into undefined
          },
        },
        flowFile: '/data/flows.json',
        userDir: '/data/node-red-home',
        nodesDir: '/data/nodes',
        editorTheme: {
          page: {
            title: 'My Node-RED',
            favicon: '/img/favicon.png',
          },
          header: {
            title: 'Production',
            image: '/img/logo.png',
            url: 'https://example.com',
          },
          deployButton: {
            type: 'simple',
            label: 'Push',
            icon: 'rocket',
          },
          palette: {
            editable: true,
            catalogues: [
              'https://catalogue1.example.com',
              'https://catalogue2.example.com',
            ],
          },
          projects: {
            enabled: true,
          },
          codeEditor: {
            lib: 'monaco',
            options: {
              theme: 'vs-dark',
              fontSize: 14,
            },
          },
          userMenu: true,
          tours: true,
          login: {
            image: '/img/login.png',
          },
          logout: {
            redirect: 'https://logout.example.com',
          },
        },
        runtimeState: {
          enabled: true,
          file: '/data/runtime-state.json',
        },
        lang: 'es-ES',
        adminAuth: {
          users: [
            {
              username: 'admin',
            },
          ],
        },
      };

      const result = configToFormData(completeConfig);

      expect(result.uiPort).toBe(1880);
      expect(result.uiHost).toBe('192.168.1.100');
      expect(result.httpAdminRoot).toBe('/admin');
      expect(result.httpNodeRoot).toBe('/api');
      expect(result.disableEditor).toBe(false);
      expect(result.projectsEnabled).toBe(true);
      expect(result.loggingConsoleLevel).toBe('debug');
      expect(result.loggingConsoleMetrics).toBe(true);
      expect(result.loggingInternalLevel).toBe('info');
      expect(result.loggingInternalMetrics).toBe(true);
      expect(result.flowFile).toBe('/data/flows.json');
      expect(result.userDir).toBe('/data/node-red-home');
      expect(result.nodesDir).toBe('/data/nodes');
      expect(result.editorPageTitle).toBe('My Node-RED');
      expect(result.editorPageFavicon).toBe('/img/favicon.png');
      expect(result.editorHeaderTitle).toBe('Production');
      expect(result.editorHeaderImage).toBe('/img/logo.png');
      expect(result.editorHeaderUrl).toBe('https://example.com');
      expect(result.editorDeployType).toBe('simple');
      expect(result.editorDeployLabel).toBe('Push');
      expect(result.editorDeployIcon).toBe('rocket');
      expect(result.editorPaletteEditable).toBe(true);
      expect(result.editorPaletteCatalogues).toBe(
        'https://catalogue1.example.com\nhttps://catalogue2.example.com'
      );
      expect(result.editorProjectsEnabled).toBe(true);
      expect(result.editorCodeLib).toBe('monaco');
      expect(result.editorCodeTheme).toBe('vs-dark');
      expect(result.editorCodeFontSize).toBe(14);
      expect(result.editorUserMenu).toBe(true);
      expect(result.editorTours).toBe(true);
      expect(result.editorLoginImage).toBe('/img/login.png');
      expect(result.editorLogoutRedirect).toBe('https://logout.example.com');
      expect(result.runtimeStateEnabled).toBe(true);
      expect(result.runtimeStateFile).toBe('/data/runtime-state.json');
      expect(result.lang).toBe('es-ES');
      expect(result.authEnabled).toBe(true);
      expect(result.authAdminUser).toBe('admin');
      expect(result.authAdminPassword).toBe(''); // never expose password hash
    });

    it('should provide sensible defaults for minimal config', () => {
      const minimalConfig = {
        uiPort: 1880,
        projectsEnabled: false,
      };

      const result = configToFormData(minimalConfig);

      expect(result.uiPort).toBe(1880);
      expect(result.uiHost).toBe('0.0.0.0');
      expect(result.httpAdminRoot).toBe('/');
      expect(result.httpNodeRoot).toBe('/');
      expect(result.disableEditor).toBe(false);
      expect(result.projectsEnabled).toBe(false);
      expect(result.loggingConsoleLevel).toBe('info');
      expect(result.loggingConsoleMetrics).toBe(false);
      expect(result.loggingInternalLevel).toBe('info');
      expect(result.loggingInternalMetrics).toBe(false);
      expect(result.flowFile).toBe('flows.json');
      expect(result.userDir).toBe('');
      expect(result.nodesDir).toBe('');
      expect(result.editorPageTitle).toBe('Node-RED');
      expect(result.editorHeaderTitle).toBe('Node-RED');
      expect(result.lang).toBe('en-US');
      expect(result.authEnabled).toBe(false);
    });

    it('should handle authentication enabled state correctly', () => {
      const configWithAuth = {
        uiPort: 1880,
        projectsEnabled: false,
        adminAuth: {
          users: [
            {
              username: 'testuser',
            },
          ],
        },
      };

      const result = configToFormData(configWithAuth);

      expect(result.authEnabled).toBe(true);
      expect(result.authAdminUser).toBe('testuser');
      expect(result.authAdminPassword).toBe(''); // empty: don't expose hash
    });

    it('should handle multiple authentication types', () => {
      const configWithMultiAuth = {
        uiPort: 1880,
        projectsEnabled: false,
        adminAuth: {
          users: [
            {
              username: 'admin-user',
            },
          ],
        },
        nodeHttpAuth: {
          users: [
            {
              username: 'http-user',
            },
          ],
        },
        staticAuth: {
          users: [
            {
              username: 'static-user',
            },
          ],
        },
      };

      const result = configToFormData(configWithMultiAuth);

      expect(result.authEnabled).toBe(true);
      expect(result.authAdminUser).toBe('admin-user');
      expect(result.authNodeHttpEnabled).toBe(true);
      expect(result.authNodeHttpUser).toBe('http-user');
      expect(result.authStaticEnabled).toBe(true);
      expect(result.authStaticUser).toBe('static-user');
    });

    it('should handle editor theme with only page title', () => {
      const config = {
        uiPort: 1880,
        projectsEnabled: false,
        editorTheme: {
          page: {
            title: 'Custom Title',
          },
        },
      };

      const result = configToFormData(config);

      expect(result.editorPageTitle).toBe('Custom Title');
      expect(result.editorPageFavicon).toBe('');
      expect(result.editorPageCss).toBe('');
    });

    it('should handle palette catalogues as array joining with newlines', () => {
      const config = {
        uiPort: 1880,
        projectsEnabled: false,
        editorTheme: {
          palette: {
            catalogues: ['url1', 'url2', 'url3'],
          },
        },
      };

      const result = configToFormData(config);

      expect(result.editorPaletteCatalogues).toBe('url1\nurl2\nurl3');
    });

    it('should handle ace code editor defaults', () => {
      const config = {
        uiPort: 1880,
        projectsEnabled: false,
        editorTheme: {
          codeEditor: {
            lib: 'ace',
            options: {
              theme: 'vs',
            },
          },
        },
      };

      const result = configToFormData(config);

      expect(result.editorCodeLib).toBe('ace');
      expect(result.editorCodeTheme).toBe('vs');
      expect(result.editorCodeFontSize).toBe(12);
    });

    it('should handle disabled editor features', () => {
      const config = {
        uiPort: 1880,
        projectsEnabled: false,
        editorTheme: {
          userMenu: false,
          tours: false,
        },
      };

      const result = configToFormData(config);

      expect(result.editorUserMenu).toBe(false);
      expect(result.editorTours).toBe(false);
    });

    it('should handle runtime state settings', () => {
      const config = {
        uiPort: 1880,
        projectsEnabled: false,
        runtimeState: {
          enabled: true,
          file: '/custom/runtime-state.json',
        },
      };

      const result = configToFormData(config);

      expect(result.runtimeStateEnabled).toBe(true);
      expect(result.runtimeStateFile).toBe('/custom/runtime-state.json');
    });

    it('should handle missing authentication users gracefully', () => {
      const config = {
        uiPort: 1880,
        projectsEnabled: false,
        adminAuth: {
          users: [
            {
              username: 'admin',
            },
          ],
        },
        nodeHttpAuth: {
          users: [], // empty array
        },
      };

      const result = configToFormData(config);

      expect(result.authAdminUser).toBe('admin');
      expect(result.authNodeHttpUser).toBe('');
      expect(result.authNodeHttpEnabled).toBe(true); // enabled because nodeHttpAuth exists
    });

    it('should handle editor palette editable default (true when not specified)', () => {
      const config = {
        uiPort: 1880,
        projectsEnabled: false,
        editorTheme: {
          palette: {
            catalogues: ['url1'],
          },
        },
      };

      const result = configToFormData(config);

      expect(result.editorPaletteEditable).toBe(true); // defaults to true
    });

    it('should handle editor palette editable false', () => {
      const config = {
        uiPort: 1880,
        projectsEnabled: false,
        editorTheme: {
          palette: {
            editable: false,
            catalogues: ['url1'],
          },
        },
      };

      const result = configToFormData(config);

      expect(result.editorPaletteEditable).toBe(false);
    });
  });

  // ============================================
  // formDataToConfigPayload Tests
  // ============================================

  describe('formDataToConfigPayload', () => {
    it('should convert form data to API payload with all fields', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '192.168.1.100',
        httpAdminRoot: '/admin',
        httpNodeRoot: '/api',
        disableEditor: true,
        projectsEnabled: true,
        loggingConsoleLevel: 'debug',
        loggingConsoleMetrics: true,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: true,
        flowFile: '/data/flows.json',
        userDir: '/data/node-red-home',
        nodesDir: '/data/nodes',
        editorPageTitle: 'My Node-RED',
        editorPageFavicon: '/img/favicon.png',
        editorPageCss: 'body { color: red; }',
        editorHeaderTitle: 'Production',
        editorHeaderImage: '/img/logo.png',
        editorHeaderUrl: 'https://example.com',
        editorDeployType: 'simple',
        editorDeployLabel: 'Push',
        editorDeployIcon: 'rocket',
        editorPaletteEditable: true,
        editorPaletteCatalogues: 'https://catalogue1.example.com\nhttps://catalogue2.example.com',
        editorProjectsEnabled: true,
        editorCodeLib: 'monaco',
        editorCodeTheme: 'vs-dark',
        editorCodeFontSize: 14,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '/img/login.png',
        editorLogoutRedirect: 'https://logout.example.com',
        runtimeStateEnabled: true,
        runtimeStateFile: '/data/runtime-state.json',
        lang: 'es-ES',
        authEnabled: true,
        authAdminUser: 'admin',
        authAdminPassword: 'password123',
        authNodeHttpEnabled: true,
        authNodeHttpUser: 'http-user',
        authNodeHttpPassword: 'http-pass',
        authStaticEnabled: true,
        authStaticUser: 'static-user',
        authStaticPassword: 'static-pass',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.uiPort).toBe(1880);
      expect(result.uiHost).toBe('192.168.1.100');
      expect(result.httpAdminRoot).toBe('/admin');
      expect(result.httpNodeRoot).toBe('/api');
      expect(result.disableEditor).toBe(true);
      expect(result.projectsEnabled).toBe(true);
      expect(result.logging?.console?.level).toBe('debug');
      expect(result.logging?.console?.metrics).toBe(true);
      expect(result.logging?.internal?.level).toBe('info');
      expect(result.logging?.internal?.metrics).toBe(true);
      expect(result.flowFile).toBe('/data/flows.json');
      expect(result.userDir).toBe('/data/node-red-home');
      expect(result.nodesDir).toBe('/data/nodes');
      expect(result.lang).toBe('es-ES');
    });

    it('should create adminAuth when authEnabled and authAdminUser provided', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: true,
        authAdminUser: 'admin-user',
        authAdminPassword: 'secret123',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.adminAuth).toEqual({
        type: 'credentials',
        users: [
          {
            username: 'admin-user',
            password: 'secret123',
            permissions: '*',
          },
        ],
      });
    });

    it('should not create adminAuth when authEnabled is false', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.adminAuth).toBeUndefined();
    });

    it('should preserve empty password when authAdminPassword is empty (backend preserves hash)', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: true,
        authAdminUser: 'admin',
        authAdminPassword: '', // empty = don't change backend hash
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.adminAuth?.users?.[0]?.password).toBe('');
    });

    it('should create nodeHttpAuth when enabled with all fields', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: true,
        authNodeHttpUser: 'nodehttp-user',
        authNodeHttpPassword: 'nodehttp-pass',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.nodeHttpAuth).toEqual({
        type: 'credentials',
        users: [
          {
            username: 'nodehttp-user',
            password: 'nodehttp-pass',
          },
        ],
      });
    });

    it('should not create nodeHttpAuth when missing any required field', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: true,
        authNodeHttpUser: '', // missing!
        authNodeHttpPassword: 'pass',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.nodeHttpAuth).toBeUndefined();
    });

    it('should create editor theme with all page settings', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: 'Custom Title',
        editorPageFavicon: '/custom-favicon.ico',
        editorPageCss: 'body { margin: 0; }',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.editorTheme?.page).toEqual({
        title: 'Custom Title',
        favicon: '/custom-favicon.ico',
        css: 'body { margin: 0; }',
      });
    });

    it('should handle palette catalogues split by newlines', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: 'url1\nurl2\nurl3\n\nurl4', // with empty lines
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.editorTheme?.palette?.catalogues).toEqual([
        'url1',
        'url2',
        'url3',
        'url4',
      ]); // empty lines filtered out
    });

    it('should handle monaco code editor with custom theme', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'monaco',
        editorCodeTheme: 'vs-dark',
        editorCodeFontSize: 14,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.editorTheme?.codeEditor).toEqual({
        lib: 'monaco',
        options: { theme: 'vs-dark' },
      });
    });

    it('should not include codeEditor when using default ace with default theme', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.editorTheme?.codeEditor).toBeUndefined();
    });

    it('should handle disabled user menu and tours', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: false,
        editorTours: false,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.editorTheme?.userMenu).toBe(false);
      expect(result.editorTheme?.tours).toBe(false);
    });

    it('should handle runtime state settings', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: true,
        runtimeStateFile: '/custom/runtime.json',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.runtimeState).toEqual({
        enabled: true,
        file: '/custom/runtime.json',
      });
    });

    it('should not include runtimeState when disabled', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.runtimeState).toBeUndefined();
    });

    it('should default uiHost to 0.0.0.0 when empty', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.uiHost).toBe('0.0.0.0');
    });

    it('should default httpAdminRoot and httpNodeRoot to / when empty', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '',
        httpNodeRoot: '',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: 'flows.json',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.httpAdminRoot).toBe('/');
      expect(result.httpNodeRoot).toBe('/');
    });

    it('should default flowFile to flows.json when empty', () => {
      const formData: NodeRedConfigFormData = {
        uiPort: 1880,
        uiHost: '0.0.0.0',
        httpAdminRoot: '/',
        httpNodeRoot: '/',
        disableEditor: false,
        projectsEnabled: false,
        loggingConsoleLevel: 'info',
        loggingConsoleMetrics: false,
        loggingInternalLevel: 'info',
        loggingInternalMetrics: false,
        flowFile: '',
        userDir: '',
        nodesDir: '',
        editorPageTitle: '',
        editorPageFavicon: '',
        editorPageCss: '',
        editorHeaderTitle: '',
        editorHeaderImage: '',
        editorHeaderUrl: '',
        editorDeployType: 'default',
        editorDeployLabel: '',
        editorDeployIcon: '',
        editorPaletteEditable: true,
        editorPaletteCatalogues: '',
        editorProjectsEnabled: false,
        editorCodeLib: 'ace',
        editorCodeTheme: 'vs',
        editorCodeFontSize: 12,
        editorUserMenu: true,
        editorTours: true,
        editorLoginImage: '',
        editorLogoutRedirect: '',
        runtimeStateEnabled: false,
        runtimeStateFile: '',
        lang: 'en-US',
        authEnabled: false,
        authAdminUser: '',
        authAdminPassword: '',
        authNodeHttpEnabled: false,
        authNodeHttpUser: '',
        authNodeHttpPassword: '',
        authStaticEnabled: false,
        authStaticUser: '',
        authStaticPassword: '',
      };

      const result = formDataToConfigPayload(formData);

      expect(result.flowFile).toBe('flows.json');
    });
  });

  // ============================================
  // Round-Trip Tests
  // ============================================

  describe('round-trip conversions', () => {
    it('should maintain data integrity through configToFormData -> formDataToConfigPayload cycle', () => {
      const originalConfig = {
        uiPort: 1880,
        uiHost: '192.168.1.1',
        httpAdminRoot: '/admin',
        httpNodeRoot: '/api',
        disableEditor: false,
        projectsEnabled: true,
        logging: {
          console: { level: 'debug', metrics: true },
          internal: { level: 'info', metrics: true },
        },
        flowFile: 'flows.json',
        userDir: '/data',
        nodesDir: '/nodes',
        editorTheme: {
          page: { title: 'My App' },
          header: { title: 'Header' },
          palette: { editable: true, catalogues: ['url1', 'url2'] },
        },
        lang: 'en-US',
      };

      const formData = configToFormData(originalConfig);
      const reconstructedPayload = formDataToConfigPayload(formData);

      // Check basic fields survive the round-trip
      expect(reconstructedPayload.uiPort).toBe(originalConfig.uiPort);
      expect(reconstructedPayload.uiHost).toBe(originalConfig.uiHost);
      expect(reconstructedPayload.httpAdminRoot).toBe(originalConfig.httpAdminRoot);
      expect(reconstructedPayload.httpNodeRoot).toBe(originalConfig.httpNodeRoot);
      expect(reconstructedPayload.projectsEnabled).toBe(originalConfig.projectsEnabled);
      expect(reconstructedPayload.lang).toBe(originalConfig.lang);

      // Check logging survives
      expect(reconstructedPayload.logging?.console?.level).toBe('debug');
      expect(reconstructedPayload.logging?.console?.metrics).toBe(true);
      expect(reconstructedPayload.logging?.internal?.level).toBe('info');
      expect(reconstructedPayload.logging?.internal?.metrics).toBe(true);

      // Check catalogues survive (split and rejoin)
      expect(reconstructedPayload.editorTheme?.palette?.catalogues).toEqual([
        'url1',
        'url2',
      ]);
    });

    it('should not preserve authentication passwords through round-trip (security)', () => {
      const configWithAuth = {
        uiPort: 1880,
        projectsEnabled: false,
        adminAuth: {
          users: [
            {
              username: 'admin',
            },
          ],
        },
      };

      const formData = configToFormData(configWithAuth);

      // Password should be empty in form data (don't expose hash to frontend)
      expect(formData.authAdminPassword).toBe('');

      // When converting back, the empty password is preserved
      const payload = formDataToConfigPayload(formData);
      expect(payload.adminAuth?.users?.[0]?.password).toBe('');
    });
  });
});
