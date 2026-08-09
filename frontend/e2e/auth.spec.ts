import { expect, test } from '@playwright/test'
import { installApiMocks, login } from './helpers'

test.describe('NRCC auth boundaries', () => {
  test('authenticated session survives a page reload without another login', async ({ page }) => {
    let loginRequests = 0
    page.on('request', (request) => {
      if (request.method() === 'POST' && new URL(request.url()).pathname === '/api/auth/login') {
        loginRequests += 1
      }
    })

    await login(page)
    await page.reload()

    await expect(page).toHaveURL(/\/dashboard$/)
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
    expect(loginRequests).toBe(1)
  })

  test('unauthenticated browser landing on /dashboard is redirected to /login', async ({ page }) => {
    await installApiMocks(page)
    await page.route('**/api/auth/refresh', async (route) => {
      await route.abort('failed')
    })
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/login$/)
  })

  test('logout clears the session and protected routes redirect again afterwards', async ({ page }) => {
    await login(page)
    await page.getByRole('button', { name: /open user menu/ }).click()
    await page.getByRole('menuitem', { name: 'Sign out' }).click()
    await expect(page).toHaveURL(/\/login$/)

    await page.route('**/api/auth/refresh', async (route) => {
      await route.abort('failed')
    })
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/login$/)
  })

  test('non-admin user navigating to /settings/users is kept out of the Users page', async ({ page }) => {
    await login(page)
    // Override /auth/me AFTER login because login() re-installs the default
    // mock routes (last-registered wins in Playwright). Without this, the
    // admin mockUser would survive and the viewer gate never trips.
    await page.route('**/api/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            id: 'user-viewer',
            username: 'viewer',
            role: 'viewer',
            createdAt: '2026-01-01T00:00:00.000Z',
          },
          timestamp: new Date(0).toISOString(),
        }),
      })
    })
    // ProtectedRoute redirects non-admin users from /settings/users to
    // /dashboard (see ProtectedRoute.tsx:35). The observable signal is the
    // URL: a viewer should never see the Users page render.
    await page.goto('/settings/users')
    await expect(page).not.toHaveURL(/\/settings\/users$/)
  })
})
