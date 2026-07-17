import { expect, type Page, test } from '@playwright/test'
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

type Scenario = 'initialized' | 'setup-required'

type JsonValue = Record<string, unknown> | unknown[] | string | number | boolean | null

const envelope = (data: JsonValue) => ({ success: true, data, timestamp: new Date(0).toISOString() })

async function installApiMocks(page: Page, scenario: Scenario = 'initialized') {
  await page.route('**/api/**', async (route) => {
    const request = route.request()
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

    return json({ success: false, error: { code: 'UNMOCKED', message: `${method} ${path}` }, timestamp: new Date(0).toISOString() }, 500)
  })
}

async function login(page: Page) {
  await installApiMocks(page)
  await page.goto('/login')
  await page.getByLabel('Username').fill('admin')
  await page.getByLabel('Password').fill('password123')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
}

test.describe('NRCC smoke E2E flows with fixture API', () => {
  test('setup flow creates the first administrator without host side effects', async ({ page }) => {
    await installApiMocks(page, 'setup-required')
    await page.goto('/setup')
    await page.getByLabel('Username').fill('admin')
    await page.getByLabel('Password', { exact: true }).fill('password123')
    await page.getByLabel('Confirm Password').fill('password123')
    await page.getByRole('button', { name: 'Create account and continue' }).click()
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  })

  test('login flow opens the dashboard with representative status responses', async ({ page }) => {
    await login(page)
    await expect(page.getByText('System Health')).toBeVisible()
    await expect(page.getByText('Quick Actions')).toBeVisible()
  })

  test('Node-RED runtime start/stop smoke path is mocked and non-destructive', async ({ page }) => {
    await installApiMocks(page)
    const responses: string[] = []
    page.on('response', (response) => {
      if (response.url().includes('/api/runtime/')) responses.push(`${response.request().method()} ${new URL(response.url()).pathname}`)
    })

    await page.goto('/login')
    await page.evaluate(async () => {
      await fetch('/api/runtime/start', { method: 'POST' })
      await fetch('/api/runtime/stop', { method: 'POST' })
    })

    expect(responses).toContain('POST /api/runtime/start')
    expect(responses).toContain('POST /api/runtime/stop')
  })

  test('backup creation uses fixture responses', async ({ page }) => {
    await login(page)
    await page.getByRole('link', { name: /Backups/ }).click()
    await expect(page.getByRole('heading', { name: 'Backups', exact: true })).toBeVisible()
    await page.getByRole('button', { name: /Crear backup ahora/ }).first().click()
    await expect(page.getByRole('button', { name: 'Manual smoke backup' })).toBeVisible()
  })

  test('restore dry path requires confirmation and returns a fixture result', async ({ page }) => {
    await login(page)
    await page.goto('/backups')
    await expect(page.getByRole('button', { name: 'Manual smoke backup' })).toBeVisible()
    await page.locator('button[title="Restore"]').first().click()
    await expect(page.getByText('Restaurar backup')).toBeVisible()
    await page.locator('input[placeholder="backup-001"]').fill('backup-001')
    await expect(page.getByRole('button', { name: /Confirm/ })).toBeEnabled()
    const restoreResult = await page.evaluate(async () => {
      const response = await fetch('/api/backups/backup-001/restore', { method: 'POST' })
      return response.json()
    })
    expect(JSON.stringify(restoreResult)).toContain('Restore dry path completed in test mode')
  })

  test('critical navigation pages render with fixture API data', async ({ page }) => {
    await login(page)
    const pages = [
      { link: /Flows/, heading: 'Flows' },
      { link: /Libraries/, heading: 'npm Libraries' },
      { link: /Backups/, heading: 'Backups' },
    ]

    for (const { link, heading } of pages) {
      await page.getByRole('link', { name: link }).click()
      await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible()
    }
  })
})
