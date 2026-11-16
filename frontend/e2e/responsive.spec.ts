import { expect, test, type Page } from '@playwright/test'
import { envelope, installApiMocks, login } from './helpers'

// T12 adapt pass: every route fits the viewport at a phone (320) and a
// tablet (768) width, with no document-level horizontal scroll. The admin
// has an update available, so the header carries its widest control set.
const ROUTES = [
  '/overview',
  '/configuration',
  '/security',
  '/environment',
  '/backups',
  '/bootstrap',
  '/profile',
  '/settings/users',
  '/maintenance/updates',
  '/maintenance/libraries',
]

async function mockUpdateAvailable(page: Page) {
  await page.route(/\/api\/updates\/(status|check)$/, (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify(envelope({
        status: 'update-available',
        updateAvailable: true,
        currentVersion: '1.0.0',
        latestVersion: '1.1.0',
        releaseNotes: null,
        checkedAt: '2026-01-01T00:00:00.000Z',
      })),
    }),
  )
}

async function expectNoHorizontalOverflow(page: Page, label: string) {
  const { scrollWidth, clientWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }))
  expect(scrollWidth, `${label}: document scrollWidth ${scrollWidth} > clientWidth ${clientWidth}`).toBeLessThanOrEqual(clientWidth)
}

for (const width of [320, 768]) {
  test.describe(`responsive at ${width}px`, () => {
    test.use({ viewport: { width, height: 900 } })

    test('public routes fit the viewport', async ({ page }) => {
      await installApiMocks(page, 'setup-required')
      await page.goto('/setup')
      await expect(page.getByRole('button', { name: 'Create account and continue' })).toBeVisible()
      await expectNoHorizontalOverflow(page, '/setup')

      await page.unrouteAll()
      await installApiMocks(page)
      await page.goto('/login')
      await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
      await expectNoHorizontalOverflow(page, '/login')
    })

    test('authenticated routes fit the viewport', async ({ page }) => {
      await login(page)
      await mockUpdateAvailable(page)
      for (const route of ROUTES) {
        await page.goto(route)
        await expect(page.getByRole('heading', { level: 1 }).first()).toBeVisible()
        if (route === '/overview') await expect(page.getByRole('button', { name: /update available/i })).toBeVisible()
        await expectNoHorizontalOverflow(page, route)
      }
    })
  })
}
