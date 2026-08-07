import { expect, test } from '@playwright/test'
import { installApiMocks, login } from './helpers'

test.describe('NRCC auth boundaries', () => {
  test('unauthenticated browser landing on /dashboard is redirected to /login', async ({ page }) => {
    await installApiMocks(page)
    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/login$/)
  })

  test('logout clears the session and protected routes redirect again afterwards', async ({ page }) => {
    await login(page)
    await page.locator('[data-testid="app-sidebar"]').getByRole('button').first().click()
    await page.getByRole('menuitem', { name: 'Sign out' }).click()
    await expect(page).toHaveURL(/\/login$/)

    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/login$/)
  })
})
