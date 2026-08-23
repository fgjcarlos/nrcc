import { expect, type Page } from '@playwright/test'
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
} from '../src/test/msw/fixtures'

export type Scenario = 'initialized' | 'setup-required'
export type PostureScenario = 'healthy' | 'degraded' | 'critical'

export type JsonValue = Record<string, unknown> | unknown[] | string | number | boolean | null

export const envelope = (data: JsonValue) => ({ success: true, data, timestamp: new Date(0).toISOString() })

export async function installApiMocks(page: Page, scenario: Scenario = 'initialized', posture: PostureScenario = 'healthy') {
  const securityPosture = {
    healthy: { encryptionKeyConfigured: true, backupDownloadAdminOnly: true, activeRefreshSessions: 0, mfa: { enrolledAdmins: 2, totalAdmins: 2 } },
    degraded: { encryptionKeyConfigured: true, backupDownloadAdminOnly: true, activeRefreshSessions: 3, mfa: { enrolledAdmins: 1, totalAdmins: 2 } },
    critical: { encryptionKeyConfigured: false, backupDownloadAdminOnly: false, activeRefreshSessions: 0, mfa: { enrolledAdmins: 0, totalAdmins: 2 } },
  };
  await page.route(/\/api\//, async (route, request) => {
    // Only intercept calls to the actual backend API. Vite-dev serves ESM
    // modules under paths like /src/shared/api/index.ts which also contain
    // "/api/" — never route those through the mock handler.
    if (!new URL(request.url()).pathname.startsWith('/api/')) {
      await route.continue()
      return
    }

    const url = new URL(request.url())
    const path = url.pathname.replace('/api', '')
    const method = request.method()

    const json = (data: JsonValue, status = 200) =>
      route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(data) })

    if (method === 'GET' && path === '/auth/status') {
      return json(envelope(scenario === 'setup-required' ? authStatusSetupRequired : authStatusInitialized))
    }
    if (method === 'POST' && ['/auth/setup', '/auth/login'].includes(path)) return json(envelope(authResponse))
    if (method === 'POST' && path === '/auth/refresh') return json(envelope({ token: authResponse.token }))
    if (method === 'POST' && path === '/auth/logout') return json(envelope({ message: 'Logged out' }))
    if (method === 'GET' && path === '/auth/me') return json(envelope(mockUser))
    if (method === 'GET' && path === '/auth/users') return json(envelope([mockUser]))

    if (method === 'GET' && path === '/bootstrap/status') return json(envelope(hostStatus))
    if (method === 'GET' && path === '/system/info') return json(envelope(systemInfo))
    if (method === 'GET' && path === '/system/security-posture') return json(envelope(securityPosture[posture]))
    if (method === 'GET' && path === '/system/history') return json(envelope([]))
    if (method === 'GET' && path === '/docker/status') {
      return json(envelope({
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
      }))
    }
    if (method === 'GET' && path === '/config') return json(envelope({ uiPort: 1880, uiHost: '127.0.0.1', projectsEnabled: true }))
    if (method === 'GET' && path === '/config/default') return json(envelope({ uiPort: 1880, uiHost: '127.0.0.1', projectsEnabled: true }))
    if (method === 'GET' && path === '/settings/raw') return json(envelope({ content: '// fixture settings.js for e2e', backupPath: '/tmp/nrcc-smoke', writable: true }))
    if (method === 'GET' && path === '/runtime/history') return json(envelope({ events: [], status: { status: 'running', uptime: 0, restartCount: 0, consecutiveFailures: 0 } }))
    if (method === 'POST' && ['/runtime/start', '/runtime/stop', '/runtime/restart'].includes(path)) {
      return json(envelope({ message: 'Node-RED runtime action handled in fixture test mode' }))
    }

    if (method === 'GET' && path === '/backups/status') return json(envelope(backupSchedulerStatus))
    if (method === 'GET' && path === '/backups/observability') return json(envelope(backupObservability))
    if (method === 'GET' && path === '/backups/config') return json(envelope(backupConfig))
    if (method === 'POST' && path === '/backups/config') return json(envelope(backupConfig))
    if (method === 'GET' && path === '/backups/storage') return json(envelope(backupObservability.storage))
    if (method === 'GET' && path === '/backups') {
      return json(envelope(url.searchParams.has('page')
        ? { items: backups, total: backups.length, page: 1, limit: 10 }
        : backups))
    }
    if (method === 'POST' && path === '/backups') return json(envelope({ ...backups[0], id: 'backup-smoke-created', name: 'Manual smoke backup created' }))
    if (method === 'GET' && path === '/backups/backup-001') return json(envelope(backupManifest))
    if (method === 'POST' && path === '/backups/backup-001/restore') {
      return json(envelope({ success: true, message: 'Restore dry path completed in test mode', preRestoreId: 'pre-restore-001' }))
    }

    if (method === 'GET' && path === '/libraries') return json(envelope(libraries))
    if (method === 'GET' && path === '/flows') return json(envelope(flows))
    if (method === 'GET' && path === '/files') return json(envelope([{ path: 'flows.json', type: 'file', size: 1024 }]))

    // Coverage for the previously unmocked routes (issue #565):
    // /env, /env/dotenv, /updates/state, /updates/status, /updates/check,
    // /updates/history. Each is the smallest shape the consuming page needs.
    if (method === 'GET' && path === '/env') return json(envelope([]))
    if (method === 'GET' && path === '/env/dotenv') return json(envelope({ content: 'NRCC_TEST=1\n', path: '/tmp/nrcc-smoke/.env', exists: true }))
    if (method === 'POST' && path === '/env/import-from-node-red') return json(envelope({ lines: [], issues: [], valid: true, summary: 'No env vars imported in test mode' }))
    if (method === 'GET' && path === '/updates/state') return json(envelope({ phase: 'idle', currentVersion: '1.0.0', targetVersion: null, progress: 0, startedAt: null, finishedAt: null, error: null }))
    if (method === 'GET' && path === '/updates/status') return json(envelope({ status: 'up-to-date', currentVersion: '1.0.0', latestVersion: '1.0.0', releaseNotes: null, checkedAt: '2026-01-01T00:00:00.000Z' }))
    if (method === 'GET' && path === '/updates/check') return json(envelope({ status: 'up-to-date', currentVersion: '1.0.0', latestVersion: '1.0.0', releaseNotes: null, checkedAt: '2026-01-01T00:00:00.000Z' }))
    if (method === 'GET' && path === '/updates/history') return json(envelope([]))

    return json({ success: false, error: { code: 'UNMOCKED', message: `${method} ${path}` }, timestamp: new Date(0).toISOString() }, 500)
  })
}

export async function login(page: Page, posture: PostureScenario = 'healthy') {
  await installApiMocks(page, 'initialized', posture)
  await page.goto('/login')
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('password123')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
}
