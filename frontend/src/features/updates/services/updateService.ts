import { api } from 'shared/lib/api';

export interface UpdateStatus {
  currentVersion: string;
  latestVersion: string;
  updateAvailable: boolean;
  checkedAt: string;
  error?: string;
}

export interface UpdateFlowState {
  state: 'Idle' | 'BackingUp' | 'Applying' | 'Completed' | 'Failed';
  phase: string;
  backupId?: string;
  error?: string;
  availableVersion?: string;
}

export interface UpdateHistoryEntry {
  id: string;
  timestamp: string;
  fromVersion: string;
  toVersion: string;
  appliedBy: string;
  status: 'success' | 'error';
  errorMessage?: string;
}

export interface UpdateApplyResponse {
  success: boolean;
  message: string;
  fromVersion?: string;
  toVersion?: string;
}

export const updateService = {
  getFlowState: async (): Promise<UpdateFlowState> => {
    try {
      const response = await api.get<{ data: UpdateFlowState }>('/updates/state');
      return response.data.data ?? {
        state: 'Idle',
        phase: 'idle',
      };
    } catch (error) {
      return {
        state: 'Idle',
        phase: 'idle',
        error: error instanceof Error ? error.message : 'Failed to fetch flow state',
      };
    }
  },

  getStatus: async (): Promise<UpdateStatus> => {
    try {
      const response = await api.get<{ data: UpdateStatus }>('/updates/status');
      return response.data.data ?? {
        currentVersion: 'unknown',
        latestVersion: 'unknown',
        updateAvailable: false,
        checkedAt: new Date().toISOString(),
      };
    } catch (error) {
      // Return error state to allow page to render gracefully
      return {
        currentVersion: 'unknown',
        latestVersion: 'unknown',
        updateAvailable: false,
        checkedAt: new Date().toISOString(),
        error: error instanceof Error ? error.message : 'Failed to fetch status',
      };
    }
  },

  check: async (): Promise<UpdateStatus> => {
    try {
      const response = await api.get<{ data: UpdateStatus }>('/updates/check');
      return response.data.data ?? {
        currentVersion: 'unknown',
        latestVersion: 'unknown',
        updateAvailable: false,
        checkedAt: new Date().toISOString(),
      };
    } catch (error) {
      return {
        currentVersion: 'unknown',
        latestVersion: 'unknown',
        updateAvailable: false,
        checkedAt: new Date().toISOString(),
        error: error instanceof Error ? error.message : 'Failed to check for updates',
      };
    }
  },

  applyUpdate: async (): Promise<UpdateApplyResponse> => {
    try {
      const response = await api.post<{ data: UpdateApplyResponse }>('/updates/apply');
      return response.data.data ?? {
        success: false,
        message: 'No response from server',
      };
    } catch (error) {
      return {
        success: false,
        message: error instanceof Error ? error.message : 'Failed to apply update',
      };
    }
  },

  getHistory: async (): Promise<UpdateHistoryEntry[]> => {
    try {
      const response = await api.get<{ data: UpdateHistoryEntry[] }>('/updates/history');
      return response.data.data ?? [];
    } catch {
      // Return empty array on error to keep page stable
      return [];
    }
  },
};
