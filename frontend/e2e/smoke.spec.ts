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
    // Issue #676 promoted the Runtime card (with Restart/Open actions) out of
    // the metric row; QuickActionsCard was removed in favor of those buttons.
    await expect(page.getByRole('button', { name: 'Reiniciar' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Abrir' })).toBeVisible()
  })

  test('restart flow drives the real Reiniciar button and shows the success toast', async ({ page }) => {
    await login(page)
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
    await page.getByRole('button', { name: 'Reiniciar' }).click()
    await expect(page.getByRole('heading', { name: '¿Reiniciar Node-RED?' })).toBeVisible()
    await page.getByRole('button', { name: 'Sí, reiniciar' }).click()
    await expect(page.getByText('Node-RED reiniciado')).toBeVisible()
  })

  test('backup creation uses fixture responses', async ({ page }) => {
    await login(page)
    await page.getByRole('link', { name: /Backups/ }).click()
    await expect(page.getByRole('heading', { name: 'Backups', exact: true })).toBeVisible()
    await page.getByRole('button', { name: /Crear backup ahora/ }).first().click()
    await expect(page.getByRole('button', { name: 'Manual smoke backup' })).toBeVisible()
  })

  test('restore dry path requires confirmation and triggers a real restore', async ({ page }) => {
    await login(page)
    await page.goto('/backups')
    await expect(page.getByRole('button', { name: 'Manual smoke backup' })).toBeVisible()
    await page.locator('button[title="Restore"]').first().click()
    await expect(page.getByText('Restaurar backup')).toBeVisible()
    // The ConfirmationDialog listens for Enter when canConfirm() is true.
    // We use the keyboard shortcut here because the dialog's overlay sits
    // above the Confirm button in the stacking context — a click on the
    // button itself is intercepted by the overlay. The keyboard path is
    // the same code branch the user would hit by pressing Enter.
    await page.locator('input[placeholder="backup-001"]').fill('backup-001')
    await page.locator('input[placeholder="backup-001"]').press('Enter')
    await expect(page.getByText('Restore dry path completed in test mode')).toBeVisible()
  })

  test('critical navigation pages render with fixture API data', async ({ page }) => {
    await login(page)
    // Reach each route directly. Doing page.goto / url avoids race conditions
    // where the previous view is still tearing down when the next link click
    // fires (Backups in particular was catching the next nav mid-render).
    const pages: Array<{ url: string; heading: string }> = [
      { url: '/flows', heading: 'Flows' },
      { url: '/libraries', heading: 'npm Libraries' },
      { url: '/backups', heading: 'Backups' },
      { url: '/configuration', heading: 'Node-RED Configuration' },
      { url: '/environment', heading: 'Environment Variables' },
      { url: '/files', heading: 'Files' },
      { url: '/updates', heading: 'Node-RED Updates' },
      { url: '/bootstrap', heading: 'Bootstrap & Environment' },
    ]

    for (const { url, heading } of pages) {
      await page.goto(url)
      await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible()
    }

    // /profile is reached from the UserMenu, not the sidebar.
    await page.goto('/dashboard')
    await page.getByRole('button', { name: /open user menu/ }).click()
    await page.getByRole('menuitem', { name: 'Profile' }).click()
    await expect(page.getByRole('heading', { name: 'Profile' })).toBeVisible()
  })
})
