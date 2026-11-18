import { expect, test } from '@playwright/test'
import { editableHostStatus, securityCenterConfig } from '../src/test/msw/fixtures'
import { envelope, login } from './helpers'

async function openSecurityCenter(
  page: import('@playwright/test').Page,
  apply: (body: Record<string, unknown>, count: number) => { status: number; body: unknown },
) {
  let count = 0
  await login(page)
  await page.route('**/api/bootstrap/status', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify(envelope(editableHostStatus)),
    }),
  )
  await page.route('**/api/config', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify(envelope(securityCenterConfig)),
    }),
  )
  await page.route('**/api/settings/raw', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify(
        envelope({
          content: 'module.exports = {};',
          writable: true,
          revision: { fingerprint: 'current-revision', algorithm: 'sha256' },
        }),
      ),
    }),
  )
  await page.route('**/api/config/apply', (route, request) => {
    count += 1
    const result = apply(JSON.parse(request.postData() ?? '{}'), count)
    return route.fulfill({ status: result.status, contentType: 'application/json', body: JSON.stringify(result.body) })
  })
  // Issue #766 slice E — SecurityView now renders 4 independent
  // boundary cards; replace the "Security Center" heading check
  // with one of the boundary card titles.
  await page.goto('/security')
  await expect(page.getByRole('heading', { name: 'Node-RED admin auth' })).toBeVisible()
}

// T10: each surface form sits behind its Edit disclosure; open it before filling.
async function expandSurface(page: import('@playwright/test').Page, testId: string) {
  await page.getByTestId(testId).locator('summary').click()
}

// P1 #1: Save opens the shared review drawer; its Apply sends the request.
async function saveAndApply(page: import('@playwright/test').Page, testId: string) {
  await page.getByTestId(`${testId}-save`).click()
  const drawer = page.getByRole('dialog', { name: 'Review changes' })
  await expect(drawer).toBeVisible()
  await drawer.getByRole('button', { name: 'Apply and restart Node-RED' }).click()
  await expect(drawer).toBeHidden()
}

// The surface's change circuit live region carries the outcome (no toast).
const circuitLive = (page: import('@playwright/test').Page, testId: string) =>
  page.getByTestId(testId).getByTestId('change-circuit-live')

test.describe('Security Center transactional apply (issue #766 slice E — per-surface boundaries)', () => {
  test('submits each canonical authentication surface independently with a revision', async ({ page }) => {
    const requests: Record<string, unknown>[] = []
    await openSecurityCenter(page, (body) => {
      requests.push(body)
      return {
        status: 200,
        body: envelope({ document: { revision: { fingerprint: 'next-revision', algorithm: 'sha256' } } }),
      }
    })

    // AdminAuthBoundaryCard — edit the operator username and save
    // via the surface-scoped button (was "Save Security Center" in
    // slice A; now "Save admin auth").
    await expandSurface(page, 'boundary-admin-auth')
    await page.getByTestId('boundary-admin-auth-username').first().fill('operator-updated')
    await saveAndApply(page, 'boundary-admin-auth')
    await expect.poll(() => requests.length).toBe(1)
    expect(requests[0]).toMatchObject({
      expectedRevision: 'current-revision',
      adminAuth: { users: [{ username: 'operator-updated' }] },
    })
    expect(requests[0]).not.toHaveProperty('httpNodeAuth')
    expect(requests[0]).not.toHaveProperty('httpStaticAuth')
    await expect(circuitLive(page, 'boundary-admin-auth')).toHaveText('Ready: Closed. Readiness OK')

    // HttpBasicAuthBoundaryCard surface=httpNodeAuth — edit the
    // basic-auth user field and save via the per-surface button.
    await expandSurface(page, 'boundary-httpNodeAuth')
    await page.getByTestId('boundary-httpNodeAuth-user').fill('nodes-updated')
    await saveAndApply(page, 'boundary-httpNodeAuth')
    await expect.poll(() => requests.length).toBe(2)
    expect(requests[1]).toMatchObject({
      expectedRevision: 'current-revision',
      httpNodeAuth: { user: 'nodes-updated' },
    })
    expect(requests[1]).not.toHaveProperty('adminAuth')
    expect(requests[1]).not.toHaveProperty('httpStaticAuth')

    // HttpBasicAuthBoundaryCard surface=httpStaticAuth — same flow.
    await expandSurface(page, 'boundary-httpStaticAuth')
    await page.getByTestId('boundary-httpStaticAuth-user').fill('static-updated')
    await saveAndApply(page, 'boundary-httpStaticAuth')
    await expect.poll(() => requests.length).toBe(3)
    expect(requests[2]).toMatchObject({
      expectedRevision: 'current-revision',
      httpStaticAuth: { user: 'static-updated' },
    })
    expect(requests[2]).not.toHaveProperty('adminAuth')
    expect(requests[2]).not.toHaveProperty('httpNodeAuth')
  })

  test('renders clear retry guidance for stale and readiness-failure transactions', async ({ page }) => {
    await openSecurityCenter(page, (_body, count) => ({
      status: count === 1 ? 409 : 500,
      body: {
        success: false,
        error: { code: count === 1 ? 'SETTINGS_REVISION_CONFLICT' : 'APPLY_ERROR', message: 'redacted' },
      },
    }))
    // First click hits the revision-conflict branch (409 → SETTINGS_REVISION_CONFLICT).
    await expandSurface(page, 'boundary-admin-auth')
    await page.getByTestId('boundary-admin-auth-username').first().fill('operator-updated')
    await saveAndApply(page, 'boundary-admin-auth')
    // The inline error stays; the circuit opens at Validate. No toast.
    await expect(page.getByTestId('boundary-admin-auth-error')).toContainText(
      'Settings changed elsewhere. Refresh, review the redacted preview, then retry.',
    )
    await expect(circuitLive(page, 'boundary-admin-auth')).toHaveText('Validate: Failed. Rejected; nothing written')
    // Second apply hits the readiness-failure branch (500 → APPLY_ERROR, a code
    // without a known stage: the circuit says the outcome is not confirmed).
    await saveAndApply(page, 'boundary-admin-auth')
    await expect(page.getByTestId('boundary-admin-auth-error')).toContainText(
      'The transaction did not complete. Any failed readiness check is rolled back; review runtime readiness, then retry.',
    )
    await expect(circuitLive(page, 'boundary-admin-auth')).toHaveText(
      'Validate: Not confirmed. Outcome not confirmed; check the runtime',
    )
    await expect(page.locator('[data-sonner-toast]')).toHaveCount(0)
  })
})
