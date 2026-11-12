import { expect, test, type Page } from '@playwright/test'
import { installApiMocks, login } from './helpers'
import {
  expectPinnedTheme,
  isMobileProject,
  snapshotFocusBaseline,
  tabAndProbe,
  type FocusStop,
} from './a11y-helpers'

/**
 * a11y-keyboard.spec.ts — issue #889 (S3, S6, S9)
 *
 * Keyboard accessibility in the pinned theme of each project:
 *  - every essential control shows a visible focus indicator;
 *  - modal dialogs close with Escape, keep Tab inside and return focus;
 *  - on the mobile viewport the navigation drawer opens by keyboard and
 *    a navigation link can be reached and activated.
 *
 * Transitions are disabled (reduced motion) so computed focus styles are
 * read in their final state rather than mid-transition.
 */
test.use({ reducedMotion: 'reduce' })

async function sweepTabOrder(page: Page, maxStops: number): Promise<FocusStop[]> {
  await snapshotFocusBaseline(page)
  const stops: FocusStop[] = []
  for (let i = 0; i < maxStops; i += 1) {
    const stop = await tabAndProbe(page)
    if (!stop) break
    if (stops.length > 0 && stop.label === stops[0].label && stop.region === stops[0].region) break
    stops.push(stop)
  }
  return stops
}

function describeStops(stops: FocusStop[]): string {
  return stops.map((s) => `[${s.region}] ${s.label} — ${s.detail}`).join('\n')
}

test.describe('Issue #889 — keyboard focus visibility', () => {
  test('login controls show a visible focus indicator', async ({ page }) => {
    await installApiMocks(page, 'initialized')
    await page.goto('/login')
    await expect(page.getByLabel('Username')).toBeVisible()
    await expectPinnedTheme(page)

    const stops = await sweepTabOrder(page, 15)
    const labels = stops.map((s) => s.label)
    expect(labels).toEqual(
      expect.arrayContaining(['input "username"', 'input "password"', expect.stringMatching(/^button "Sign in/)]),
    )
    const hidden = stops.filter((s) => !s.visible)
    expect(hidden, `login controls without a visible focus indicator:\n${describeStops(hidden)}`).toHaveLength(0)
  })

  test('app shell controls show a visible focus indicator', async ({ page }) => {
    await login(page)
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible()
    await expectPinnedTheme(page)

    const stops = await sweepTabOrder(page, 200)
    const shell = stops.filter((s) => s.region !== 'main' && s.region !== 'other')
    test.info().annotations.push({
      type: 'shell-focus-stops',
      description: shell.map((s) => `[${s.region}] ${s.label}`).join(', '),
    })
    const regions = new Set(shell.map((s) => s.region))
    expect(regions.has('skip-link'), 'skip link is the first stop').toBe(true)
    expect(regions.has('header'), 'header controls are reachable').toBe(true)
    if (!isMobileProject()) {
      // The closed mobile drawer hides the sidebar; on desktop it is always open.
      expect(shell.some((s) => s.region === 'sidebar' && /^a "?/.test(s.label)), 'sidebar nav links are reachable').toBe(true)
    }

    // Page-content controls are outside the shell contract; record them for
    // follow-up instead of failing the shell assertion.
    const contentHidden = stops.filter((s) => (s.region === 'main' || s.region === 'other') && !s.visible)
    if (contentHidden.length > 0) {
      test.info().annotations.push({ type: 'content-focus-gaps', description: describeStops(contentHidden) })
    }

    const hidden = shell.filter((s) => !s.visible)
    expect(hidden, `app shell controls without a visible focus indicator:\n${describeStops(hidden)}`).toHaveLength(0)
  })
})

test.describe('Issue #889 — modal keyboard behaviour', () => {
  test('confirmation dialog: Escape closes, Tab stays inside, focus returns to the trigger', async ({ page }) => {
    await login(page)
    await page.goto('/backups')
    const trigger = page.locator('button[title="Restore"]').first()
    await trigger.focus()
    await page.keyboard.press('Enter')

    const dialog = page.getByRole('dialog', { name: 'Restaurar backup' })
    await expect(dialog).toBeVisible()
    await expect(dialog.locator('input[placeholder="backup-001"]')).toBeFocused()

    for (let i = 0; i < 6; i += 1) {
      await page.keyboard.press(i % 2 === 0 ? 'Tab' : 'Shift+Tab')
      await page.keyboard.press('Tab')
      expect(await dialog.evaluate((el) => el.contains(document.activeElement)), `Tab ${i} stays inside`).toBe(true)
    }

    await page.keyboard.press('Escape')
    await expect(dialog).toBeHidden()
    await expect(trigger).toBeFocused()
  })

  test('restart modal: initial focus on Cancel, Escape closes, Tab stays inside, focus returns', async ({ page }) => {
    await login(page)
    const trigger = page.getByRole('button', { name: 'Restart', exact: true })
    await trigger.focus()
    await page.keyboard.press('Enter')

    const dialog = page.getByRole('dialog', { name: 'Restart Node-RED?' })
    await expect(dialog).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeFocused()

    for (let i = 0; i < 4; i += 1) {
      await page.keyboard.press('Tab')
      expect(await dialog.evaluate((el) => el.contains(document.activeElement)), `Tab ${i} stays inside`).toBe(true)
    }
    await page.keyboard.press('Shift+Tab')
    expect(await dialog.evaluate((el) => el.contains(document.activeElement))).toBe(true)

    await page.keyboard.press('Escape')
    await expect(dialog).toBeHidden()
    await expect(trigger).toBeFocused()
  })
})

test.describe('Issue #889 — mobile navigation by keyboard', () => {
  test('drawer opens with the keyboard and a nav link can be reached and activated', async ({ page }) => {
    test.skip(!isMobileProject(), 'drawer navigation only applies below the lg breakpoint')

    await login(page)
    await page.reload()
    await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible()
    await expectPinnedTheme(page)

    const nav = page.getByRole('navigation', { name: 'Navegación principal' })
    await expect(nav).toBeHidden()

    // Reach the drawer toggle from the top of the page.
    let toggleReached = false
    for (let i = 0; i < 5 && !toggleReached; i += 1) {
      await page.keyboard.press('Tab')
      toggleReached = await page.evaluate(() => document.activeElement?.id === 'sidebar-drawer')
    }
    expect(toggleReached, 'drawer toggle is reachable with Tab').toBe(true)
    await page.keyboard.press('Space')
    await expect(nav).toBeVisible()

    // Tab forward until focus enters the drawer navigation.
    let tabs = 0
    let inNav = false
    while (tabs < 150 && !inNav) {
      await page.keyboard.press('Tab')
      tabs += 1
      inNav = await nav.evaluate((el) => el.contains(document.activeElement))
    }
    test.info().annotations.push({ type: 'drawer-tabs-to-nav', description: String(tabs) })
    expect(inNav, `a nav link is reachable after ${tabs} Tab presses`).toBe(true)

    const configurationLink = nav.getByRole('link', { name: 'Configuration' })
    while (!(await configurationLink.evaluate((el) => el === document.activeElement)) && tabs < 160) {
      await page.keyboard.press('Tab')
      tabs += 1
    }
    await expect(configurationLink).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/configuration$/)
  })
})
