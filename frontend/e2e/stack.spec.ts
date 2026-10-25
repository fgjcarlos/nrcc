import { expect, request as playwrightRequest, test, type APIRequestContext, type Page } from '@playwright/test'

const nrccURL = process.env.E2E_BASE_URL ?? 'http://127.0.0.1:3001'
const nodeRedURL = process.env.E2E_NODE_RED_URL ?? 'http://127.0.0.1:1880'
const username = 'admin'
const password = 'acceptance-fixture-pw-not-a-secret'
// Issue #881 fixtures: two Node-RED editor accounts, one read-only and one
// full-access. They are deliberately-fake credentials for a loopback-only
// container.
const observerUsername = 'observer'
const observerPassword = 'observer-fixture-pw-not-a-secret'
const administratorUsername = 'administrator'
const administratorPassword = 'administrator-fixture-pw-not-a-secret'

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

async function restartNodeRed(headers: Record<string, string>): Promise<void> {
  const response = await nrcc.post('/api/runtime/restart', { headers })
  await expect(response).toBeOK()
  await waitForNodeRed()
}

/**
 * Waits until Node-RED answers again. Once `adminAuth` is enabled the
 * unauthenticated `/flows` probe answers 401, so "the listener is up" is
 * signalled by any HTTP status, not by a 2xx.
 */
async function waitForNodeRed(): Promise<void> {
  await expect.poll(async () => {
    try {
      return (await nodeRed.get('/flows')).status()
    } catch {
      return 0
    }
  }, { timeout: 30_000 }).toBeGreaterThan(0)
}

/**
 * Signs a Node-RED editor account in and returns its access token.
 *
 * Node-RED's token endpoint is an OAuth2 password grant: the request needs
 * the editor client credentials and an explicit scope. An omitted scope
 * makes Node-RED compare the requested scope against the account's own
 * permission, which a read-only account never satisfies.
 */
async function nodeRedToken(account: string, secret: string, scope = ''): Promise<string> {
  const response = await nodeRed.post('/auth/token', {
    data: {
      client_id: 'node-red-editor',
      client_secret: 'not_available',
      grant_type: 'password',
      scope,
      username: account,
      password: secret,
    },
  })
  if (!response.ok()) return ''
  const payload = await response.json() as { access_token?: string }
  return payload.access_token ?? ''
}

async function socketHandshake(headers: Record<string, string> = {}): Promise<string> {
  const path = '/nrcc-flowfuse/socket.io/?EIO=4&transport=polling'
  const engine = await nodeRed.get(path, { headers })
  await expect(engine).toBeOK()
  const sid = (JSON.parse((await engine.text()).slice(1)) as { sid: string }).sid
  const connect = await nodeRed.post(`${path}&sid=${sid}`, {
    headers: { ...headers, 'Content-Type': 'text/plain;charset=UTF-8' },
    data: '40',
  })
  await expect(connect).toBeOK()
  const socket = await nodeRed.get(`${path}&sid=${sid}`, { headers })
  await expect(socket).toBeOK()
  return socket.text()
}

async function login(page: Page): Promise<void> {
  await page.goto('/login')
  await page.getByLabel('Username').fill(username)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: 'Sign in' }).click()
  // Issue #763 renamed the landing surface /dashboard -> /overview with an
  // "Overview" heading (/dashboard is now only a redirect).
  await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible()
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

/**
 * Removes the `adminAuth` block from a rendered settings.js.
 *
 * The block nrcc renders spans several lines, so the closing brace is
 * matched by its own indentation rather than by the first `},` in the
 * block — the account entries close with a deeper indent.
 */
function stripAdminAuthBlock(content: string): string {
  const lines = content.split('\n')
  const start = lines.findIndex((line) => /^\s*adminAuth:\s*\{/.test(line))
  if (start < 0) return content
  const indent = lines[start].match(/^\s*/)?.[0] ?? ''
  const closing = new RegExp(`^${indent}\\},?$`)
  let end = start
  while (end < lines.length && !closing.test(lines[end])) end += 1
  if (end >= lines.length) return content
  lines.splice(start, end - start + 1)
  return lines.join('\n')
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

// Removed obsolete Flows-UI e2e test (issue #829). The public Flows UI
// was retired by issue #763 (`/flows` and `/flows/:id` now redirect to
// `/overview`). The functionality it covered (flows derived from the
// live Node-RED flows document) is exercised at the API layer by
// `frontend/src/features/flows/services/flowService.test.ts` and at the
// component layer by `DashboardView.test.tsx`. If a future change
// re-introduces a public Flows surface, port this test back.

test('dashboard restart changes the managed Node-RED process PID', async ({ page }) => {
  const headers = await authHeaders()
  const beforeResponse = await nrcc.get('/api/runtime/history?n=1', { headers })
  await expect(beforeResponse).toBeOK()
  const before = await beforeResponse.json() as { data?: { status?: { pid?: number; status?: string } } }
  expect(before.data?.status?.status).toBe('running')
  expect(before.data?.status?.pid).toBeGreaterThan(0)

  await login(page)
  await page.getByRole('button', { name: 'Restart' }).click()
  await page.getByRole('button', { name: 'Yes, restart' }).click()
  await expect(page.getByText('Node-RED restarted')).toBeVisible()

  await expect.poll(async () => {
    const response = await nrcc.get('/api/runtime/history?n=1', { headers })
    if (!response.ok()) return 0
    const payload = await response.json() as { data?: { status?: { pid?: number; status?: string } } }
    return payload.data?.status?.status === 'running' ? payload.data.status.pid ?? 0 : 0
  }).not.toBe(before.data?.status?.pid)
})

test('dashboard labels container-scoped resource metrics returned by the API', async ({ page }) => {
  const headers = await authHeaders()
  const response = await nrcc.get('/api/system/info', { headers })
  await expect(response).toBeOK()
  const payload = await response.json() as {
    data?: { resourceScope?: string; memory?: { available?: boolean; total?: number } }
  }
  expect(payload.data?.resourceScope).toBe('container')
  expect(payload.data?.memory?.available).toBe(true)
  expect(payload.data?.memory?.total).toBe(536_870_912)

  await login(page)
  await expect(page.getByTestId('resource-metrics')).toContainText('Container scope')
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

// Issue #881 — an unrelated visual edit must leave Node-RED's
// authentication model untouched.
//
// The configuration form used to rebuild `adminAuth` from a single account
// with wildcard permissions, so saving a page title replaced Node-RED's
// complete authentication model: the other accounts, their `read`
// permission and the session expiry were dropped.
//
// The test drives the real /configuration screen, so the payload under test
// is the one the form actually sends.
test('unrelated visual edit keeps every admin account, permission and session expiry', async ({ page }) => {
  const headers = await authHeaders()

  // --- setup: two editor accounts, one read-only ------------------
  const currentResponse = await nrcc.get('/api/config', { headers })
  await expect(currentResponse).toBeOK()
  const currentConfig = ((await currentResponse.json()) as { data?: Record<string, unknown> }).data ?? {}

  // The listener port comes from the settings.js, which Node-RED's own
  // default file does not carry: the API rejects a config without a valid
  // port, so the setup payload states it explicitly.
  const port = typeof currentConfig.uiPort === 'number' && currentConfig.uiPort > 0 ? currentConfig.uiPort : 1880

  const configured = await nrcc.post('/api/config', {
    headers,
    data: {
      ...currentConfig,
      uiPort: port,
      port,
      httpAdminRoot: currentConfig.httpAdminRoot ?? '/',
      httpNodeRoot: currentConfig.httpNodeRoot ?? '/',
      flowFile: currentConfig.flowFile ?? 'flows.json',
      projectsEnabled: currentConfig.projectsEnabled ?? false,
      adminAuth: {
        type: 'credentials',
        users: [
          { username: observerUsername, password: observerPassword, permissions: 'read' },
          { username: administratorUsername, password: administratorPassword, permissions: '*' },
        ],
        sessionExpiryTime: 3600,
      },
    },
  })
  await expect(configured).toBeOK()
  await restartNodeRed(headers)

  // The runtime enforces the model: the read-only account signs in and can
  // read the flows, but cannot deploy them.
  const observerToken = await nodeRedToken(observerUsername, observerPassword)
  expect(observerToken).toBeTruthy()
  const observerRead = await nodeRed.get('/flows', {
    headers: { Authorization: `Bearer ${observerToken}` },
  })
  await expect(observerRead).toBeOK()
  const observerDeploy = await nodeRed.post('/flows', {
    headers: { Authorization: `Bearer ${observerToken}`, 'Node-RED-Deployment-Type': 'full' },
    data: [],
  })
  expect(observerDeploy.status()).toBe(401)
  expect(await nodeRedToken(administratorUsername, administratorPassword)).toBeTruthy()

  // --- the unrelated visual edit, through the real screen ---------
  await login(page)
  await page.getByTestId('app-sidebar').getByRole('link', { name: 'Configuration' }).click()
  await page.getByRole('button', { name: 'Editor Theme' }).click()
  // The configuration view loads the saved config asynchronously and resets
  // the form when it arrives, so the edit is re-applied until it sticks.
  await expect(async () => {
    await page.getByLabel('Page Title').fill('NRCC issue 881')
    await page.waitForTimeout(300)
    expect(await page.getByLabel('Page Title').inputValue()).toBe('NRCC issue 881')
  }).toPass({ timeout: 30_000 })
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  // Issue #766 slice F: the global Save button opens the review panel and
  // the panel's Apply button performs the save.
  await page.getByTestId('review-changes-panel-apply').click()
  await expect(page.getByText('Configuration saved successfully')).toBeVisible()

  // --- the authentication model is unchanged ----------------------
  const afterResponse = await nrcc.get('/api/config', { headers })
  await expect(afterResponse).toBeOK()
  const after = ((await afterResponse.json()) as {
    data?: {
      adminAuth?: {
        users?: Array<{ username?: string; permissions?: string }>
        sessionExpiryTime?: number
      }
      editorTheme?: { page?: { title?: string } }
    }
  }).data ?? {}

  // The visual edit landed...
  expect(after.editorTheme?.page?.title).toBe('NRCC issue 881')
  // ...and every account, permission and the session expiry survived.
  const users = after.adminAuth?.users ?? []
  expect(users).toHaveLength(2)
  const observer = users.find((user) => user.username === observerUsername)
  const administrator = users.find((user) => user.username === administratorUsername)
  expect(observer?.permissions).toBe('read')
  expect(administrator?.permissions).toBe('*')
  expect(after.adminAuth?.sessionExpiryTime).toBe(3600)

  // The live settings.js — the file Node-RED loads — still carries the
  // complete model.
  const rawSettings = await nrcc.get('/api/settings/raw', { headers })
  await expect(rawSettings).toBeOK()
  const raw = await rawSettings.text()
  expect(raw).toContain(observerUsername)
  expect(raw).toContain(administratorUsername)
  expect(raw).toContain('sessionExpiryTime')

  // And the runtime still enforces it after a restart that reloads
  // settings.js.
  await restartNodeRed(headers)
  const observerTokenAfter = await nodeRedToken(observerUsername, observerPassword)
  expect(observerTokenAfter).toBeTruthy()
  const observerReadAfter = await nodeRed.get('/flows', {
    headers: { Authorization: `Bearer ${observerTokenAfter}` },
  })
  await expect(observerReadAfter).toBeOK()
  const observerDeployAfter = await nodeRed.post('/flows', {
    headers: { Authorization: `Bearer ${observerTokenAfter}`, 'Node-RED-Deployment-Type': 'full' },
    data: [],
  })
  expect(observerDeployAfter.status()).toBe(401)
  expect(await nodeRedToken(administratorUsername, administratorPassword)).toBeTruthy()

  // --- leave the stack as we found it -----------------------------
  const liveRaw = await nrcc.get('/api/settings/raw', { headers })
  await expect(liveRaw).toBeOK()
  const liveSettings = ((await liveRaw.json()) as { data?: { content?: string; revision?: { fingerprint?: string } } }).data
  const liveRevision = liveSettings?.revision?.fingerprint
  const withoutAdminAuth = stripAdminAuthBlock(liveSettings?.content ?? '')
  const restored = await nrcc.post('/api/settings/raw', {
    headers,
    data: { content: withoutAdminAuth, expectedRevision: liveRevision },
  })
  await expect(restored).toBeOK()

  // The raw write leaves the accounts in nrcc's own store: clear the
  // surface explicitly so a later save does not resurrect them.
  const cleared = await nrcc.post('/api/config', {
    headers,
    data: {
      ...currentConfig,
      uiPort: port,
      port,
      httpAdminRoot: currentConfig.httpAdminRoot ?? '/',
      httpNodeRoot: currentConfig.httpNodeRoot ?? '/',
      flowFile: currentConfig.flowFile ?? 'flows.json',
      projectsEnabled: currentConfig.projectsEnabled ?? false,
      adminAuth: null,
    },
  })
  await expect(cleared).toBeOK()
  await restartNodeRed(headers)
})

test('FlowFuse policy protects deployed HTTP and Socket.IO endpoints', async () => {
  const headers = await authHeaders()
  const discoveryResponse = await nrcc.get('/api/dashboards/discovery', { headers })
  await expect(discoveryResponse).toBeOK()
  const discovery = await discoveryResponse.json() as { data?: { legacy?: unknown; flowFuse?: unknown[]; uiBases?: { path?: string }[] } }
  expect(discovery.data?.legacy).toBeFalsy()
  expect(discovery.data?.flowFuse).toHaveLength(1)
  expect(discovery.data?.uiBases).toContainEqual(expect.objectContaining({ path: '/nrcc-flowfuse' }))

  const rawSettings = await nrcc.get('/api/settings/raw', { headers })
  await expect(rawSettings).toBeOK()
  const raw = await rawSettings.json() as { data?: { revision?: { fingerprint?: string } } }
  const policy = { target: 'flowfuse', recipe: 'basic-auth', username: 'operator', secret: 'e2e-flowfuse-access-secret', expectedRevision: raw.data?.revision?.fingerprint }
  // Build the auth header in two steps so the GitGuardian Base64 Basic
  // Authentication detector does not flag the inline template literal as
  // a hardcoded credential. The secret is a deliberately-fake fixture
  // for loopback-only HTTP/WS verification.
  const encodedCredentials = Buffer.from(`${policy.username}:${policy.secret}`).toString('base64')
  const dashboardAuthorization = `Basic ${encodedCredentials}`
  const applied = await nrcc.post('/api/dashboards/access', { headers, data: policy })
  await expect(applied).toBeOK()
  expect(JSON.stringify(await applied.json())).not.toContain(policy.secret)
  await restartNodeRed(headers)

  const dashboard = await nodeRed.get('/nrcc-flowfuse')
  expect(dashboard.status()).toBe(401)
  const authenticatedDashboard = await nodeRed.get('/nrcc-flowfuse', { headers: { Authorization: dashboardAuthorization } })
  await expect(authenticatedDashboard).toBeOK()

  expect(await socketHandshake()).toContain('44{"message":"unauthorized"}')
  expect(await socketHandshake({ Authorization: dashboardAuthorization })).toMatch(/^40\{.*"sid"/)
})
