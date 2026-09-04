import { expect, request as playwrightRequest, test, type APIRequestContext, type Page } from '@playwright/test'

const nrccURL = process.env.E2E_BASE_URL ?? 'http://127.0.0.1:3001'
const nodeRedURL = process.env.E2E_NODE_RED_URL ?? 'http://127.0.0.1:1880'
const username = 'admin'
const password = 'acceptance-fixture-pw-not-a-secret'

let nrcc: APIRequestContext
let nodeRed: APIRequestContext

async function ensureAdmin(): Promise<void> {
  const status = await nrcc.get('/api/auth/status')
  await expect(status).toBeOK()
  const payload = await status.json() as { data?: { initialized?: boolean } }
  if (payload.data?.initialized === false) {
    const setup = await nrcc.post('/api/auth/setup', { data: { username, password } })
    await expect(setup).toBeOK()
  }
}

async function authHeaders(): Promise<Record<string, string>> {
  const response = await nrcc.post('/api/auth/login', { data: { username, password } })
  await expect(response).toBeOK()
  const payload = await response.json() as { data?: { token?: string } }
  expect(payload.data?.token).toBeTruthy()
  return { Authorization: `Bearer ${payload.data?.token}` }
}

async function login(page: Page): Promise<void> {
  await page.goto('/login')
  await page.getByLabel('Username').fill(username)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
}

async function seedProbeFlow(): Promise<void> {
  await expect.poll(async () => {
    try {
      return (await nodeRed.get('/flows')).ok()
    } catch {
      return false
    }
  }).toBe(true)

  const response = await nodeRed.post('/flows', {
    headers: { 'Node-RED-Deployment-Type': 'full' },
    data: [
      { id: 'nrcc-e2e-tab', type: 'tab', label: 'NRCC E2E Flow', disabled: false, info: '' },
      {
        id: 'nrcc-e2e-global-config',
        type: 'global-config',
        env: [{ name: 'NODE_RED_E2E_IMPORTED', value: 'available-on-entry', type: 'str' }],
        modules: {},
      },
      {
        id: 'nrcc-e2e-http-in',
        type: 'http in',
        z: 'nrcc-e2e-tab',
        name: 'Environment probe',
        url: '/nrcc-e2e-env',
        method: 'get',
        upload: false,
        swaggerDoc: '',
        wires: [['nrcc-e2e-function']],
      },
      {
        id: 'nrcc-e2e-function',
        type: 'function',
        z: 'nrcc-e2e-tab',
        name: 'Read NRCC env',
        func: 'msg.payload = env.get("NRCC_E2E_PROBE") || "missing"; return msg;',
        outputs: 1,
        timeout: 0,
        noerr: 0,
        initialize: '',
        finalize: '',
        libs: [],
        wires: [['nrcc-e2e-http-out']],
      },
      {
        id: 'nrcc-e2e-http-out',
        type: 'http response',
        z: 'nrcc-e2e-tab',
        name: '',
        statusCode: '',
        headers: {},
        wires: [],
      },
    ],
  })
  await expect(response).toBeOK()
}

test.describe.configure({ mode: 'serial' })

test.beforeAll(async () => {
  nrcc = await playwrightRequest.newContext({ baseURL: nrccURL })
  nodeRed = await playwrightRequest.newContext({ baseURL: nodeRedURL })
  await ensureAdmin()
})

test.afterAll(async () => {
  await nrcc.dispose()
  await nodeRed.dispose()
})

test('unknown API routes fail as JSON instead of returning the SPA', async () => {
  const response = await nrcc.get('/api/this-route-must-not-exist')
  expect(response.status()).toBe(404)
  expect(response.headers()['content-type']).toContain('application/json')
  const payload = await response.json() as { error?: { code?: string } }
  expect(payload.error?.code).toBe('API_ROUTE_NOT_FOUND')
})

test('flow list and detail are derived from the live Node-RED flows document', async ({ page }) => {
  await seedProbeFlow()
  await login(page)
  await page.getByRole('link', { name: 'Flows', exact: true }).click()

  await expect(page.getByRole('heading', { name: 'Flows' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'NRCC E2E Flow' })).toBeVisible()
  await expect(page.getByText('3 nodes')).toBeVisible()
  await expect(page.getByText('2 connections')).toBeVisible()

  await page.getByRole('link', { name: 'NRCC E2E Flow' }).click()
  await expect(page.getByRole('heading', { name: 'NRCC E2E Flow' })).toBeVisible()
  await expect(page.getByText('http in (1)')).toBeVisible()
  await expect(page.getByText('function (1)')).toBeVisible()
  await expect(page.getByText('http response (1)')).toBeVisible()
})

test('dashboard restart changes the managed Node-RED process PID', async ({ page }) => {
  const headers = await authHeaders()
  const beforeResponse = await nrcc.get('/api/runtime/history?n=1', { headers })
  await expect(beforeResponse).toBeOK()
  const before = await beforeResponse.json() as { data?: { status?: { pid?: number; status?: string } } }
  expect(before.data?.status?.status).toBe('running')
  expect(before.data?.status?.pid).toBeGreaterThan(0)

  await login(page)
  await page.getByRole('button', { name: 'Reiniciar' }).click()
  await page.getByRole('button', { name: 'Sí, reiniciar' }).click()
  await expect(page.getByText('Node-RED reiniciado')).toBeVisible()

  await expect.poll(async () => {
    const response = await nrcc.get('/api/runtime/history?n=1', { headers })
    if (!response.ok()) return 0
    const payload = await response.json() as { data?: { status?: { pid?: number; status?: string } } }
    return payload.data?.status?.status === 'running' ? payload.data.status.pid ?? 0 : 0
  }).not.toBe(before.data?.status?.pid)
})

test('environment action reaches a live Node-RED flow after its restart', async ({ page }) => {
  await seedProbeFlow()
  await login(page)
  await page.getByRole('link', { name: 'Environment', exact: true }).click()
  await page.getByRole('button', { name: 'Add' }).click()
  await page.getByPlaceholder('MY_VARIABLE').fill('NRCC_E2E_PROBE')
  await page.getByTestId('env-var-value-field').locator('input').fill('propagated-through-node-red')
  await page.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByText('Variable created successfully')).toBeVisible()

  await expect.poll(async () => {
    try {
      const response = await nodeRed.get('/nrcc-e2e-env')
      return response.ok() ? await response.text() : ''
    } catch {
      return ''
    }
  }, { timeout: 30_000 }).toBe('propagated-through-node-red')
})

test('environment page imports a Node-RED global entry automatically on route entry', async ({ page }) => {
  await seedProbeFlow()
  await login(page)
  await page.getByRole('link', { name: 'Environment', exact: true }).click()

  await expect(page.getByText('NODE_RED_E2E_IMPORTED')).toBeVisible()
  await expect(page.getByRole('cell', { name: 'Node-RED', exact: true })).toBeVisible()
  await expect(page.getByTestId('node-red-sync-status')).toContainText('synchronized')
})
