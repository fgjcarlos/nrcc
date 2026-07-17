import { http, HttpResponse, type HttpHandler } from 'msw'
import {
  authResponse,
  authStatusInitialized,
  authStatusSetupRequired,
  backupConfig,
  backupManifest,
  backupObservability,
  backupSchedulerStatus,
  backups,
  flows,
  hostStatus,
  libraries,
  mockUser,
  systemInfo,
} from './fixtures'

const ok = <T>(data: T) =>
  HttpResponse.json({ success: true, data, timestamp: new Date(0).toISOString() })

export type NrccMswScenario = 'initialized' | 'setup-required'

export function createNrccApiHandlers(scenario: NrccMswScenario = 'initialized'): HttpHandler[] {
  const authStatus = scenario === 'setup-required' ? authStatusSetupRequired : authStatusInitialized

  return [
    http.get('/api/auth/status', () => ok(authStatus)),
    http.post('/api/auth/setup', async () => ok(authResponse)),
    http.post('/api/auth/login', async () => ok(authResponse)),
    http.post('/api/auth/refresh', () => ok({ token: authResponse.token })),
    http.post('/api/auth/logout', () => ok({ message: 'Logged out' })),
    http.get('/api/auth/me', () => ok(mockUser)),
    http.get('/api/auth/users', () => ok([mockUser])),

    http.get('/api/bootstrap/status', () => ok(hostStatus)),
    http.get('/api/system/info', () => ok(systemInfo)),
    http.get('/api/docker/status', () => ok({
      success: true,
      data: {
        container: {
          id: 'nrcc-smoke',
          name: 'nrcc-smoke-node-red',
          image: 'nodered/node-red:4.1',
          status: 'running',
          created: '2026-01-01T00:00:00.000Z',
          ports: [{ privatePort: 1880, publicPort: 1880, type: 'tcp' }],
          state: { running: true, paused: false, restartCount: 0, memory: 128_000_000, cpu: 2 },
        },
        inDocker: false,
      },
      timestamp: new Date(0).toISOString(),
    })),
    http.get('/api/config', () => ok({ uiPort: 1880, uiHost: '127.0.0.1', projectsEnabled: true })),
    http.get('/api/runtime/history', () => ok({ events: [], status: { status: 'running', uptime: 0, restartCount: 0, consecutiveFailures: 0 } })),
    http.post('/api/runtime/start', () => ok({ message: 'Node-RED start requested in test mode' })),
    http.post('/api/runtime/stop', () => ok({ message: 'Node-RED stop requested in test mode' })),
    http.post('/api/runtime/restart', () => ok({ message: 'Node-RED restart requested in test mode' })),
    http.get('/api/runtime/logs', () => ok([{ id: 'log-1', timestamp: new Date(0).toISOString(), level: 'info', message: 'Fixture log entry' }])),

    http.get('/api/backups/status', () => ok(backupSchedulerStatus)),
    http.get('/api/backups/observability', () => ok(backupObservability)),
    http.get('/api/backups/config', () => ok(backupConfig)),
    http.post('/api/backups/config', async ({ request }) => ok({ ...backupConfig, ...(await request.json() as object) })),
    http.get('/api/backups/storage', () => ok(backupObservability.storage)),
    http.get('/api/backups', ({ request }) => {
      const url = new URL(request.url)
      if (url.searchParams.has('page')) {
        return ok({ items: backups, total: backups.length, page: 1, limit: 10, totalPages: 1 })
      }
      return ok(backups)
    }),
    http.post('/api/backups', async () => ok({ ...backups[0], id: 'backup-smoke-created', name: 'Manual smoke backup created' })),
    http.get('/api/backups/:id', () => ok(backupManifest)),
    http.post('/api/backups/:id/restore', () => ok({ success: true, message: 'Restore dry path completed in test mode', preRestoreId: 'pre-restore-001' })),
    http.delete('/api/backups/:id', () => ok({ message: 'Deleted in test mode' })),

    http.get('/api/libraries', () => ok(libraries)),
    http.post('/api/libraries/search', () => ok([])),
    http.post('/api/libraries/install', () => ok({ message: 'Install skipped in test mode', installed: true })),
    http.get('/api/libraries/:name/check', ({ params }) => ok({ name: String(params.name), installed: true })),
    http.delete('/api/libraries/:name', () => ok({ message: 'Uninstall skipped in test mode' })),

    http.get('/api/flows', () => ok(flows)),
    http.get('/api/files', () => ok([{ path: 'flows.json', type: 'file', size: 1024 }])),
  ]
}

export const handlers = createNrccApiHandlers()
