import api from '@/shared/lib';
import type { NodeRedConfig, ApiResponse, SettingsDocument } from '@/shared/types';

/**
 * Per-request timeout for saves that write settings.js and restart Node-RED.
 *
 * The backend answers only after the restart passed its readiness probe:
 * up to 6 attempts of 5 s each plus backoff (internal/service/apply_readiness.go,
 * defaultReadinessTimeout x defaultReadinessAttempts = 30 s), and a rollback
 * restart on failure. The 10 s axios default is shorter than that, and an
 * aborted request cancels the transaction server-side, so a healthy save is
 * rolled back and reported as a failure. Read and validation requests keep
 * the default.
 */
export const CONFIG_APPLY_TIMEOUT_MS = 120_000;

export const configService = {
  getConfig: () => api.get<ApiResponse<NodeRedConfig>>('/config'),

  /**
   * Save the structured configuration.
   *
   * Issue #878: `expectedRevision` is the settings.js fingerprint the
   * operator read before composing the edit. The backend rejects the save
   * with 409 SETTINGS_REVISION_CONFLICT when the live revision moved on,
   * so a stale browser session cannot overwrite newer state. It is
   * optional for clients that never learned the revision.
   */
  updateConfig: (config: Record<string, unknown>, expectedRevision?: string) =>
    api.post<ApiResponse<NodeRedConfig>>(
      '/config',
      expectedRevision ? { ...config, expectedRevision } : config,
      { timeout: CONFIG_APPLY_TIMEOUT_MS },
    ),

  applyConfig: (config: Record<string, unknown>, expectedRevision: string) =>
    api.post<ApiResponse<{ configuration: NodeRedConfig; document: SettingsDocument }>>('/config/apply', {
      ...config,
      expectedRevision,
    }, { timeout: CONFIG_APPLY_TIMEOUT_MS }),

  validateConfig: (config: Partial<NodeRedConfig>) => 
    api.post<ApiResponse<{ valid: boolean; errors: string[] }>>('/config/validate', config),
  
  getDefaultConfig: () => 
    api.get<ApiResponse<NodeRedConfig>>('/config/default'),
};

export const fileService = {
  uploadImage: (type: 'favicon' | 'header' | 'login', file: File) => {
    const formData = new FormData();
    formData.append('file', file, `${type}-${file.name}`);
    
    return api.post<ApiResponse<{ path: string; filename: string }>>('/files/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 2 * 60_000,
    });
  },
  
  deleteImage: (path: string) => {
    return api.delete<ApiResponse<{ deleted: boolean }>>(`/files/${encodeURIComponent(path)}`);
  },
  
};

export const settingsService = {
  getRaw: () => api.get<ApiResponse<SettingsDocument>>('/settings/raw'),
  /**
   * Save the raw settings.js source.
   *
   * Issue #878: `expectedRevision` is the fingerprint of the document the
   * operator opened. The backend refuses the write with 409 when the file
   * moved on, so a stale editor session cannot overwrite newer source.
   */
  saveRaw: (content: string, expectedRevision?: string) =>
    api.post<ApiResponse<{ message: string }>>('/settings/raw', {
      content,
      ...(expectedRevision ? { expectedRevision } : {}),
    }, { timeout: CONFIG_APPLY_TIMEOUT_MS }),
};
export { aiService } from './aiService';
export type { AIConfig, AIConfigInput, AIProviderStatus } from './aiService';
