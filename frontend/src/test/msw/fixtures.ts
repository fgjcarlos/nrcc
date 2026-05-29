import type { AuthStatus, User } from '@/features/auth/services/authService'
import type { BackupConfig, BackupManifest, BackupObservability, BackupSchedulerStatus, BackupSummary } from '@/features/backups/services'
import type { HostStatus, SystemInfo } from '@/shared/types'

export const mockUser: User = {
  id: 'user-admin',
  username: 'admin',
  role: 'admin',
  createdAt: '2026-01-01T00:00:00.000Z',
}

export const authStatusInitialized: AuthStatus = { initialized: true }
export const authStatusSetupRequired: AuthStatus = { initialized: false }

export const authResponse = {
  token: 'nrcc-test-token',
  user: mockUser,
}

export const hostStatus: HostStatus = {
  platform: 'linux',
  ready: true,
  interactive: false,
  nodejs: { name: 'node', installed: true, version: 'v22.0.0', command: 'node' },
  npm: { name: 'npm', installed: true, version: '11.0.0', command: 'npm' },
  nodeRedBinary: { name: 'node-red', installed: true, version: '4.0.0', command: 'node-red' },
  docker: { name: 'docker', installed: true, version: '27.0.0', command: 'docker' },
  dockerCompose: { name: 'docker compose', installed: true, version: '2.30.0', command: 'docker compose' },
  nodeRed: {
    detected: true,
    mode: 'native',
    managedByNrcc: true,
    running: true,
    version: '4.0.0',
    executable: '/usr/bin/node-red',
    userDir: '/tmp/nrcc-smoke',
    settingsPath: '/tmp/nrcc-smoke/settings.js',
  },
  settings: {
    path: '/tmp/nrcc-smoke/settings.js',
    source: 'fixture',
    writable: true,
    backupPath: '/tmp/nrcc-smoke/backups',
  },
  recommendations: [],
}

export const systemInfo: SystemInfo = {
  cpu: { usage: 14, cores: 4 },
  memory: { total: 8_589_934_592, used: 2_147_483_648, free: 6_442_450_944, usagePercent: 25 },
  disk: { total: 107_374_182_400, used: 21_474_836_480, free: 85_899_345_920, usagePercent: 20 },
  uptime: 3600,
  platform: 'linux',
  hostname: 'nrcc-smoke-host',
}

export const backupSchedulerStatus: BackupSchedulerStatus = {
  enabled: true,
  scheduled: true,
  schedule: 'daily',
  customSchedule: '',
  activeSpec: '0 2 * * *',
  nextRunAt: '2026-01-02T02:00:00.000Z',
  lastRunAt: '2026-01-01T02:00:00.000Z',
  lastSuccessAt: '2026-01-01T02:00:00.000Z',
  lastBackupId: 'backup-001',
}

export const backupConfig: BackupConfig = {
  enabled: true,
  schedule: 'daily',
  customSchedule: '',
  retentionManual: 10,
  retentionAuto: 30,
  retentionPreRestore: 5,
  includeConfig: true,
  includeSettings: true,
  includeFlowsCred: true,
  includePackageJson: true,
}

export const backups: BackupSummary[] = [
  {
    id: 'backup-001',
    name: 'Manual smoke backup',
    type: 'manual',
    createdAt: '2026-01-01T12:00:00.000Z',
    triggeredBy: 'Playwright smoke fixture',
    fileCount: 4,
    totalSize: 4096,
  },
]

export const backupManifest: BackupManifest = {
  id: 'backup-001',
  name: 'Manual smoke backup',
  type: 'manual',
  createdAt: '2026-01-01T12:00:00.000Z',
  triggeredBy: 'Playwright smoke fixture',
  totalSize: 4096,
  files: [
    { path: 'flows.json', size: 1024, checksum: 'sha256:test-flows' },
    { path: 'settings.js', size: 2048, checksum: 'sha256:test-settings' },
  ],
}

export const backupObservability: BackupObservability = {
  scheduler: backupSchedulerStatus,
  storage: {
    totalBackups: backups.length,
    totalSize: backups.reduce((total, backup) => total + backup.totalSize, 0),
    manualCount: 1,
    autoCount: 0,
    preRestoreCount: 0,
  },
  latestBackup: backups[0],
  recentEvents: [
    {
      id: 'event-001',
      type: 'manual-create',
      status: 'success',
      occurredAt: '2026-01-01T12:00:00.000Z',
      backupId: 'backup-001',
      backupName: 'Manual smoke backup',
      backupType: 'manual',
      message: 'Fixture backup created without touching the host filesystem.',
    },
  ],
}

export const libraries = [
  { name: 'node-red-dashboard', version: '3.6.5', installed: true, type: 'node' },
]

export const flows = {
  available: true,
  flows: [
    {
      id: 'flow-001',
      label: 'Smoke Flow',
      type: 'tab',
      disabled: false,
      nodeCount: 2,
    },
  ],
}
