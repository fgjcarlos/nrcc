import { expect, test } from '@playwright/test'
import { installApiMocks, login } from './helpers'

test.describe('NRCC auth boundaries', () => {
  test('unauthenticated browser landing on /dashboard is redirected to /login', async ({ page }) => {
    await installApiMocks(page)
    // Block /auth/refresh so the rehydrate path can't silently re-auth the
    // session. Without this, the default mock returns a token and
    // ProtectedRoute lets the user through.
    await page.route('**/api/auth/refresh', async (route) => {
      await route.abort('failed')
    })
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/login$/)
  })

  test('logout clears the session and protected routes redirect again afterwards', async ({ page }) => {
    await login(page)
    // The UserMenu trigger carries `aria-label="{username} — open user menu"`.
    await page.getByRole('button', { name: /open user menu/ }).click()
    await page.getByRole('menuitem', { name: 'Sign out' }).click()
    await expect(page).toHaveURL(/\/login$/)

    // Block refresh so the rehydrate path on the next /dashboard navigation
    // can't silently re-auth from the still-warm token.
    await page.route('**/api/auth/refresh', async (route) => {
      await route.abort('failed')
    })
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/login$/)
  })
})
