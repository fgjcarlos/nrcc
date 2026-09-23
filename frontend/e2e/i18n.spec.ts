import { expect, test } from '@playwright/test'
import { login } from './helpers'

test.describe('locale selection', () => {
  test('persists Spanish and keeps routed screens in the selected locale', async ({ page }) => {
    // /login is the public entry route. Log in through it so the authenticated
    // shell (which owns LocaleSwitcher) is available for the route checks.
    await login(page)

    const switcher = page.getByTestId('locale-switcher')
    await expect(switcher).toBeVisible()
    await switcher.getByRole('button', { name: 'Switch to ES' }).click()
    await expect(page.locator('html')).toHaveAttribute('lang', 'es')
    await expect(page.getByText('Reiniciar')).toBeVisible()

    await page.reload()
    await expect(page.locator('html')).toHaveAttribute('lang', 'es')
    await expect(switcher.getByRole('button', { name: 'Active: ES' })).toBeVisible()

    const routes = [
      '/login',
      '/backups',
      '/configuration',
      '/files',
      '/flows',
      '/maintenance/libraries',
      '/maintenance/updates',
    ]
    for (const route of routes) {
      await page.goto(route)
      await expect(page.locator('html')).toHaveAttribute('lang', 'es')
    }

    await page.goto('/overview')
    await switcher.getByRole('button', { name: 'Switch to EN' }).click()
    await expect(page.locator('html')).toHaveAttribute('lang', 'en')
  })
})
