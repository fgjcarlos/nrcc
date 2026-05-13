import api from '@/shared/lib';
import type { PaginationParams, PaginatedResponse } from '../types';

export type BackupType = 'manual' | 'auto' | 'pre-restore';
export type BackupSchedule = 'disabled' | 'hourly' | 'every6h' | 'daily' | 'weekly' | 'custom';

type ApiEnvelope<T> = {
  data?: T;
};

type BackupApiModel = {
  id?: unknown;
  name?: unknown;
  type?: unknown;
  createdAt?: unknown;
  triggeredBy?: unknown;
  fileCount?: unknown;
  sizeBytes?: unknown;
};

type BackupEventApiModel = {
  id?: unknown;
  type?: unknown;
  status?: unknown;
  occurredAt?: unknown;
  backupId?: unknown;
  backupName?: unknown;
  backupType?: unknown;
  message?: unknown;
  schedule?: unknown;
  activeSpec?: unknown;
  trigger?: unknown;
  prunedCount?: unknown;
  prunedIds?: unknown;
  error?: unknown;
};

type BackupObservabilityApiModel = {
  scheduler?: Record<string, unknown>;
  storage?: Record<string, unknown>;
  latestBackup?: BackupApiModel | null;
  recentEvents?: BackupEventApiModel[];
};

type BackupConfigApiModel = {
  enabled?: unknown;
  schedule?: unknown;
  scheduleInterval?: unknown;
  customSchedule?: unknown;
  retentionManual?: unknown;
  retentionAuto?: unknown;
  retentionPreRestore?: unknown;
  includeConfig?: unknown;
  includeSettings?: unknown;
  includeFlowsCred?: unknown;
  includePackageJson?: unknown;
  maxBackups?: unknown;
};

const validSchedules: BackupSchedule[] = ['disabled', 'hourly', 'every6h', 'daily', 'weekly', 'custom'];

function unwrapData<T>(payload: T | ApiEnvelope<T>): T {
  if (payload && typeof payload === 'object' && 'data' in payload) {
    return (payload as ApiEnvelope<T>).data as T;
  }

  return payload as T;
}

function toNumber(value: unknown, fallback = 0): number {
  const parsed = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function toBoolean(value: unknown, fallback: boolean): boolean {
  return typeof value === 'boolean' ? value : fallback;
}

function toStringValue(value: unknown): string | null {
  return typeof value === 'string' && value.trim().length > 0 ? value.trim() : null;
}

function normalizeBackupType(value: unknown, fallbackName?: string | null): BackupType {
  if (value === 'manual' || value === 'auto' || value === 'pre-restore') {
    return value;
  }

  return inferBackupType(fallbackName ?? null);
}

function inferBackupType(name: string | null): BackupType {
  const normalizedName = name?.toLowerCase() ?? '';

  if (normalizedName.includes('pre-restore') || normalizedName.includes('prerestore')) {
    return 'pre-restore';
  }

  if (normalizedName.includes('auto')) {
    return 'auto';
  }

  return 'manual';
}

function normalizeBackup(raw: BackupApiModel): BackupSummary {
	const id = toStringValue(raw.id) ?? 'backup-sin-id';
	const name = toStringValue(raw.name);
	const createdAt = toStringValue(raw.createdAt) ?? new Date(0).toISOString();
	const totalSize = toNumber(raw.sizeBytes, 0);
	const type = normalizeBackupType(raw.type, name);

	return {
		id,
		name: name ?? id,
		type,
		createdAt,
		triggeredBy: toStringValue(raw.triggeredBy) ?? name ?? 'Sistema',
		fileCount: Math.max(0, toNumber(raw.fileCount, 0)),
		totalSize,
  };
}

function normalizeSchedule(value: unknown): BackupSchedule {
  return typeof value === 'string' && validSchedules.includes(value as BackupSchedule)
    ? (value as BackupSchedule)
    : defaultBackupConfig.schedule;
}

function normalizeConfig(raw: BackupConfigApiModel | null | undefined): BackupConfig {
  const schedule = normalizeSchedule(raw?.schedule ?? raw?.scheduleInterval);
  const fallbackRetention = Math.max(1, toNumber(raw?.maxBackups, defaultBackupConfig.retentionManual));

  return {
    enabled: toBoolean(raw?.enabled, schedule !== 'disabled'),
    schedule,
    customSchedule: toStringValue(raw?.customSchedule) ?? '',
    retentionManual: Math.max(1, toNumber(raw?.retentionManual, fallbackRetention)),
    retentionAuto: Math.max(1, toNumber(raw?.retentionAuto, fallbackRetention)),
    retentionPreRestore: Math.max(1, toNumber(raw?.retentionPreRestore, defaultBackupConfig.retentionPreRestore)),
    includeConfig: toBoolean(raw?.includeConfig, defaultBackupConfig.includeConfig),
    includeSettings: toBoolean(raw?.includeSettings, defaultBackupConfig.includeSettings),
    includeFlowsCred: toBoolean(raw?.includeFlowsCred, defaultBackupConfig.includeFlowsCred),
    includePackageJson: toBoolean(raw?.includePackageJson, defaultBackupConfig.includePackageJson),
  };
}

function buildStorage(backups: BackupSummary[]): BackupStorageInfo {
  return backups.reduce<BackupStorageInfo>(
    (accumulator, backup) => {
      accumulator.totalBackups += 1;
      accumulator.totalSize += Number.isFinite(backup.totalSize) ? backup.totalSize : 0;

      if (backup.type === 'manual') {
        accumulator.manualCount += 1;
      } else if (backup.type === 'auto') {
        accumulator.autoCount += 1;
      } else {
        accumulator.preRestoreCount += 1;
      }

      return accumulator;
    },
    {
      totalBackups: 0,
      totalSize: 0,
      manualCount: 0,
      autoCount: 0,
      preRestoreCount: 0,
    }
  );
}

function normalizeManifest(id: string, backup: BackupSummary | null): BackupManifest {
  return {
    id,
    name: backup?.triggeredBy ?? id,
    type: backup?.type ?? 'manual',
    createdAt: backup?.createdAt ?? new Date(0).toISOString(),
    triggeredBy: backup?.triggeredBy ?? 'Sistema',
    files: [],
    totalSize: backup?.totalSize ?? 0,
  };
}

function normalizeStorage(raw: Record<string, unknown> | null | undefined): BackupStorageInfo {
  return {
    totalBackups: Math.max(0, toNumber(raw?.totalBackups, 0)),
    totalSize: Math.max(0, toNumber(raw?.totalSize, 0)),
    manualCount: Math.max(0, toNumber(raw?.manualCount, 0)),
    autoCount: Math.max(0, toNumber(raw?.autoCount, 0)),
    preRestoreCount: Math.max(0, toNumber(raw?.preRestoreCount, 0)),
  };
}

export interface BackupFileEntry {
  path: string;
  size: number;
  checksum: string;
}

export interface BackupManifest {
  id: string;
  name: string;
  type: BackupType;
  createdAt: string;
  triggeredBy: string;
  files: BackupFileEntry[];
  totalSize: number;
}

export interface BackupSummary {
	id: string;
	name: string;
	type: BackupType;
	createdAt: string;
	triggeredBy: string;
	fileCount: number;
	totalSize: number;
}

export interface BackupStorageInfo {
  totalBackups: number;
  totalSize: number;
  manualCount: number;
  autoCount: number;
  preRestoreCount: number;
}

export interface BackupSchedulerStatus {
  enabled: boolean;
  scheduled: boolean;
  schedule: BackupSchedule;
  customSchedule: string;
  activeSpec: string;
  nextRunAt: string;
  lastRunAt: string;
  lastSuccessAt: string;
  lastBackupId?: string;
  lastError?: string;
}

export type BackupEventType =
  | 'manual-create'
  | 'auto-create'
  | 'pre-restore-create'
  | 'restore'
  | 'delete'
  | 'prune'
  | 'scheduler-config'
  | 'scheduler-run'
  | 'scheduler-error';

export interface BackupEvent {
  id: string;
  type: BackupEventType;
  status: 'success' | 'error' | 'info';
  occurredAt: string;
  backupId?: string;
  backupName?: string;
  backupType?: BackupType;
  message?: string;
  schedule?: string;
  activeSpec?: string;
  trigger?: string;
  prunedCount?: number;
  prunedIds?: string[];
  error?: string;
}

export interface BackupObservability {
  scheduler: BackupSchedulerStatus;
  storage: BackupStorageInfo;
  latestBackup?: BackupSummary;
  recentEvents: BackupEvent[];
}

export interface BackupConfig {
  enabled: boolean;
  schedule: BackupSchedule;
  customSchedule: string;
  retentionManual: number;
  retentionAuto: number;
  retentionPreRestore: number;
  includeConfig: boolean;
  includeSettings: boolean;
  includeFlowsCred: boolean;
  includePackageJson: boolean;
}

export interface RestoreBackupResponse {
  success: boolean;
  message: string;
  preRestoreId?: string;
}

export const defaultBackupConfig: BackupConfig = {
  enabled: false,
  schedule: 'disabled',
  customSchedule: '',
  retentionManual: 10,
  retentionAuto: 30,
  retentionPreRestore: 5,
  includeConfig: true,
  includeSettings: true,
  includeFlowsCred: true,
  includePackageJson: true,
};

const defaultBackupSchedulerStatus: BackupSchedulerStatus = {
  enabled: false,
  scheduled: false,
  schedule: 'disabled',
  customSchedule: '',
  activeSpec: '',
  nextRunAt: '',
  lastRunAt: '',
  lastSuccessAt: '',
  lastBackupId: undefined,
  lastError: undefined,
};

function normalizeSchedulerStatus(raw: Record<string, unknown> | null | undefined): BackupSchedulerStatus {
  return {
    enabled: toBoolean(raw?.enabled, defaultBackupSchedulerStatus.enabled),
    scheduled: toBoolean(raw?.scheduled, defaultBackupSchedulerStatus.scheduled),
    schedule: normalizeSchedule(raw?.schedule),
    customSchedule: toStringValue(raw?.customSchedule) ?? '',
    activeSpec: toStringValue(raw?.activeSpec) ?? '',
    nextRunAt: toStringValue(raw?.nextRunAt) ?? '',
    lastRunAt: toStringValue(raw?.lastRunAt) ?? '',
    lastSuccessAt: toStringValue(raw?.lastSuccessAt) ?? '',
    lastBackupId: toStringValue(raw?.lastBackupId) ?? undefined,
    lastError: toStringValue(raw?.lastError) ?? undefined,
  };
}

function normalizeBackupEvent(raw: BackupEventApiModel, index: number): BackupEvent {
  const type = toStringValue(raw.type);
  const backupName = toStringValue(raw.backupName) ?? undefined;

  return {
    id: toStringValue(raw.id) ?? `backup-event-${index}`,
    type:
      type === 'manual-create' ||
      type === 'auto-create' ||
      type === 'pre-restore-create' ||
      type === 'restore' ||
      type === 'delete' ||
      type === 'prune' ||
      type === 'scheduler-config' ||
      type === 'scheduler-run' ||
      type === 'scheduler-error'
        ? type
        : 'scheduler-config',
    status: raw.status === 'success' || raw.status === 'error' || raw.status === 'info' ? raw.status : 'info',
    occurredAt: toStringValue(raw.occurredAt) ?? new Date(0).toISOString(),
    backupId: toStringValue(raw.backupId) ?? undefined,
    backupName,
    backupType: normalizeBackupType(raw.backupType, backupName) ?? undefined,
    message: toStringValue(raw.message) ?? undefined,
    schedule: toStringValue(raw.schedule) ?? undefined,
    activeSpec: toStringValue(raw.activeSpec) ?? undefined,
    trigger: toStringValue(raw.trigger) ?? undefined,
    prunedCount: Math.max(0, toNumber(raw.prunedCount, 0)) || undefined,
    prunedIds: Array.isArray(raw.prunedIds)
      ? raw.prunedIds.map((value) => toStringValue(value)).filter((value): value is string => Boolean(value))
      : undefined,
    error: toStringValue(raw.error) ?? undefined,
  };
}

function normalizeObservability(raw: BackupObservabilityApiModel | null | undefined): BackupObservability {
  return {
    scheduler: normalizeSchedulerStatus(raw?.scheduler),
    storage: normalizeStorage(raw?.storage),
    latestBackup: raw?.latestBackup ? normalizeBackup(raw.latestBackup) : undefined,
    recentEvents: Array.isArray(raw?.recentEvents) ? raw.recentEvents.map(normalizeBackupEvent) : [],
  };
}

export const backupService = {
  getStatus: async (): Promise<BackupSchedulerStatus> => {
    const response = await api.get<ApiEnvelope<Record<string, unknown>> | Record<string, unknown>>('/backups/status');
    return normalizeSchedulerStatus(unwrapData(response.data));
  },

  getObservability: async (): Promise<BackupObservability> => {
    const response = await api.get<ApiEnvelope<BackupObservabilityApiModel> | BackupObservabilityApiModel>('/backups/observability');
    return normalizeObservability(unwrapData(response.data));
  },

  getConfig: async (): Promise<BackupConfig> => {
    const response = await api.get<ApiEnvelope<BackupConfigApiModel> | BackupConfigApiModel>('/backups/config');
    return normalizeConfig(unwrapData(response.data));
  },

	saveConfig: async (config: Partial<BackupConfig>): Promise<BackupConfig> => {
		const response = await api.post<ApiEnvelope<BackupConfigApiModel> | BackupConfigApiModel>('/backups/config', config);
		return normalizeConfig(unwrapData(response.data));
	},

  list: async (): Promise<BackupSummary[]> => {
    const response = await api.get<ApiEnvelope<BackupApiModel[]> | BackupApiModel[]>('/backups');
    const rawBackups = unwrapData(response.data);

    if (!Array.isArray(rawBackups)) {
      return [];
    }

    return rawBackups.map(normalizeBackup).sort((left, right) => {
      const leftTime = new Date(left.createdAt).getTime();
      const rightTime = new Date(right.createdAt).getTime();
      return rightTime - leftTime;
    });
  },

  detail: async (id: string, backup?: BackupSummary | null): Promise<BackupManifest> => {
    try {
      const response = await api.get<ApiEnvelope<BackupManifest> | BackupManifest>(`/backups/${encodeURIComponent(id)}`);
      const manifest = unwrapData(response.data);

      if (manifest && typeof manifest === 'object' && Array.isArray(manifest.files)) {
        return {
          ...normalizeManifest(id, backup ?? null),
          ...manifest,
          name: toStringValue(manifest.name) ?? backup?.triggeredBy ?? id,
          type: normalizeBackupType(manifest.type, backup?.triggeredBy ?? null),
          createdAt: toStringValue(manifest.createdAt) ?? backup?.createdAt ?? new Date(0).toISOString(),
          triggeredBy: toStringValue(manifest.triggeredBy) ?? backup?.triggeredBy ?? 'Sistema',
          files: manifest.files
            .filter((file): file is BackupFileEntry => typeof file === 'object' && file !== null && 'path' in file)
            .map((file) => ({
              path: toStringValue(file.path) ?? 'archivo-desconocido',
              size: toNumber(file.size, 0),
              checksum: toStringValue(file.checksum) ?? 'n/d',
            })),
          totalSize: toNumber(manifest.totalSize, backup?.totalSize ?? 0),
        };
      }
    } catch {
      // Older backends do not expose backup detail. Fall back to summary-only data.
    }

    return normalizeManifest(id, backup ?? null);
  },

  create: async (type: BackupType = 'manual'): Promise<BackupSummary> => {
    const response = await api.post<ApiEnvelope<BackupApiModel> | BackupApiModel>('/backups', { type, name: type });
    return normalizeBackup(unwrapData(response.data));
  },

  restore: async (id: string): Promise<RestoreBackupResponse> => {
    const response = await api.post<ApiEnvelope<RestoreBackupResponse> | RestoreBackupResponse>(`/backups/${encodeURIComponent(id)}/restore`);
    const result = unwrapData(response.data);

    return {
      success: typeof result?.success === 'boolean' ? result.success : true,
      message: toStringValue(result?.message) ?? 'Backup restaurado correctamente',
      preRestoreId: toStringValue(result?.preRestoreId) ?? undefined,
    };
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/backups/${encodeURIComponent(id)}`);
  },

  download: async (id: string): Promise<Blob> => {
    const response = await api.get<Blob>(`/backups/${encodeURIComponent(id)}/download`, {
      responseType: 'blob',
    });
    return response.data;
  },

  getStorage: async (): Promise<BackupStorageInfo> => {
    try {
      const response = await api.get<ApiEnvelope<BackupStorageInfo> | BackupStorageInfo>('/backups/storage');
      return normalizeStorage(unwrapData(response.data));
    } catch {
      const backups = await backupService.list();
      return buildStorage(backups);
    }
  },

  listPaginated: async (
    params: PaginationParams
  ): Promise<PaginatedResponse<BackupSummary>> => {
    const searchParams = new URLSearchParams();
    searchParams.set('page', String(params.page));
    searchParams.set('limit', String(params.limit));
    if (params.sort) {
      searchParams.set('sort', params.sort);
    }
    if (params.order) {
      searchParams.set('order', params.order);
    }

    const queryString = searchParams.toString();
    const response = await api.get<
      ApiEnvelope<PaginatedResponse<BackupApiModel>> | PaginatedResponse<BackupApiModel>
    >(`/backups?${queryString}`);
    const data = unwrapData(response.data);

    return {
      items: Array.isArray(data?.items)
        ? data.items.map(normalizeBackup)
        : [],
      total: typeof data?.total === 'number' ? data.total : 0,
      page: typeof data?.page === 'number' ? data.page : params.page,
      limit: typeof data?.limit === 'number' ? data.limit : params.limit,
    };
  },

  postSchedulerConfig: async (cron: string) => {
    const response = await api.post<ApiEnvelope<{ cron: string; valid: boolean }> | { cron: string; valid: boolean }>(
      '/scheduler/config',
      { cron }
    );
    return unwrapData(response.data);
  },

  getSchedulerHistory: async (params: PaginationParams) => {
    const queryString = new URLSearchParams({
      page: String(params.page),
      limit: String(params.limit),
      ...(params.sort && { sort: params.sort }),
      ...(params.order && { order: params.order }),
    }).toString();

    const response = await api.get<
      ApiEnvelope<{
        entries: { timestamp: string; status: string; error?: string }[];
        total: number;
        page: number;
        limit: number;
      }>
    >(`/scheduler/history?${queryString}`);
    const data = unwrapData(response.data);

    return {
      entries: Array.isArray(data?.entries) ? data.entries : [],
      total: typeof data?.total === 'number' ? data.total : 0,
      page: typeof data?.page === 'number' ? data.page : params.page,
      limit: typeof data?.limit === 'number' ? data.limit : params.limit,
    };
  },

  patchStorageRetention: async (retentionDays: number) => {
    const response = await api.patch<
      ApiEnvelope<{ retentionDays: number }> | { retentionDays: number }
    >('/storage/retention', { retentionDays });
    return unwrapData(response.data);
  },
};
