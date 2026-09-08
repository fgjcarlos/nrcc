import { expect, test } from '@playwright/test'
import { editableHostStatus, securityCenterConfig } from '../src/test/msw/fixtures'
import { envelope, login } from './helpers'

async function openSecurityCenter(page: import('@playwright/test').Page, apply: (body: Record<string, unknown>, count: number) => { status: number; body: unknown }) {
  let count = 0
  await login(page)
  await page.route('**/api/bootstrap/status', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope(editableHostStatus)) }))
  await page.route('**/api/config', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope(securityCenterConfig)) }))
  await page.route('**/api/settings/raw', (route) => route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope({ content: 'module.exports = {};', writable: true, revision: { fingerprint: 'current-revision', algorithm: 'sha256' } })) }))
  await page.route('**/api/config/apply', (route, request) => {
    count += 1
    const result = apply(JSON.parse(request.postData() ?? '{}'), count)
    return route.fulfill({ status: result.status, contentType: 'application/json', body: JSON.stringify(result.body) })
  })
  await page.goto('/configuration')
  await page.getByRole('button', { name: 'Authentication' }).click()
  await expect(page.getByRole('heading', { name: 'Security Center' })).toBeVisible()
  await expect(page.getByLabel('Username').nth(0)).toHaveValue('operator')
}

test.describe('Security Center transactional apply', () => {
  test('submits each canonical authentication surface independently with a revision', async ({ page }) => {
    const requests: Record<string, unknown>[] = []
    await openSecurityCenter(page, (body) => {
      requests.push(body)
      return { status: 200, body: envelope({ document: { revision: { fingerprint: 'next-revision', algorithm: 'sha256' } } }) }
    })

    await page.getByLabel('Username').nth(0).fill('operator-updated')
    await page.getByRole('button', { name: 'Save Security Center' }).click()
    await expect.poll(() => requests.length).toBe(1)
    expect(requests[0]).toMatchObject({ expectedRevision: 'current-revision', adminAuth: { users: [{ username: 'operator-updated' }] } })
    expect(requests[0]).not.toHaveProperty('httpNodeAuth')
    expect(requests[0]).not.toHaveProperty('httpStaticAuth')

    await page.getByLabel('Username').nth(1).fill('nodes-updated')
    await page.getByRole('button', { name: 'Save Security Center' }).click()
    await expect.poll(() => requests.length).toBe(2)
    expect(requests[1]).toMatchObject({ expectedRevision: 'current-revision', httpNodeAuth: { user: 'nodes-updated' } })
    expect(requests[1]).not.toHaveProperty('adminAuth')
    expect(requests[1]).not.toHaveProperty('httpStaticAuth')

    await page.getByLabel('Username').nth(2).fill('static-updated')
    await page.getByRole('button', { name: 'Save Security Center' }).click()
    await expect.poll(() => requests.length).toBe(3)
    expect(requests[2]).toMatchObject({ expectedRevision: 'current-revision', httpStaticAuth: { user: 'static-updated' } })
    expect(requests[2]).not.toHaveProperty('adminAuth')
    expect(requests[2]).not.toHaveProperty('httpNodeAuth')
    await expect(page.getByRole('complementary', { name: 'Redacted transaction preview' })).toHaveText(/Credential values, hashes, and passwords are redacted/)
  })

  test('renders clear retry guidance for stale and readiness-failure transactions', async ({ page }) => {
    await openSecurityCenter(page, (_body, count) => ({
      status: count === 1 ? 409 : 500,
      body: { success: false, error: { code: count === 1 ? 'SETTINGS_REVISION_CONFLICT' : 'APPLY_ERROR', message: 'redacted' } },
    }))
    await page.getByLabel('Username').nth(0).fill('operator-updated')
    await page.getByRole('button', { name: 'Save Security Center' }).click()
    await expect(page.getByRole('status')).toHaveText(/Settings changed elsewhere.*Refresh.*retry/i)
    await page.getByRole('button', { name: 'Save Security Center' }).click()
    await expect(page.getByRole('status')).toHaveText(/failed readiness check is rolled back.*retry/i)
  })
})
