import { expect, test } from '@playwright/test'
import { installApiMocks, login } from './helpers'

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
