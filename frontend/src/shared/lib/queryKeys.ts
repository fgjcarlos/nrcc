import type { QueryClientConfig } from '@tanstack/react-query';

export const queryKeys = {
  auth: {
    status: ['authStatus'] as const,
    users: ['users'] as const,
  },
  backups: {
    config: ['backups-config'] as const,
    status: ['backups-status'] as const,
    observability: ['backups-observability'] as const,
    listRoot: ['backup-list'] as const,
    list: (page: number, limit: number, sort: string, order: string) =>
      ['backup-list', page, limit, sort, order] as const,
    legacyList: (page?: number, limit?: number, sort?: string, order?: string) =>
      ['backups', page, limit, sort, order] as const,
    detail: (id?: string | null) => ['backup-detail', id] as const,
    storage: ['backups-storage'] as const,
  },
  bootstrap: {
    status: ['bootstrap', 'status'] as const,
  },
  config: {
    root: ['config'] as const,
    rawSettings: ['settings', 'raw'] as const,
  },
  docker: {
    root: ['docker'] as const,
    status: ['docker', 'status'] as const,
  },
  dotenv: {
    root: ['dotenv'] as const,
  },
  envVars: {
    root: ['envVars'] as const,
    dotenv: ['envVars-dotenv'] as const,
  },
  flows: {
    root: ['flows'] as const,
    detail: (flowId?: string) => ['flow', flowId] as const,
    metrics: (flowId?: string) => ['flow-metrics', flowId] as const,
    versions: ['flow-versions'] as const,
    diff: (selectedVersions: readonly string[] | null) => ['flow-diff', selectedVersions] as const,
  },
  files: {
    root: ['files'] as const,
  },
  libraries: {
    root: ['libraries'] as const,
  },
  logs: {
    list: (levels: string[]) => ['logs', levels.join(',')] as const,
  },
  runtime: {
    status: ['runtime', 'status'] as const,
  },
  system: {
    info: ['system', 'info'] as const,
    history: ['system', 'history'] as const,
  },
  updates: {
    status: ['updateStatus'] as const,
    flowState: ['updateFlowState'] as const,
    history: ['updateHistory'] as const,
  },
} as const;

export const queryClientConfig = {
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 300_000,
      refetchOnWindowFocus: true,
      retry: 3,
    },
  },
} satisfies QueryClientConfig;
