import { expect, test, type Page } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { installApiMocks, login } from './helpers'
import { expectedTheme, expectPinnedTheme } from './a11y-helpers'

/**
 * a11y.spec.ts — issue #766 slice G W4
 *
 * Browser-level axe-core gate for the four most-trafficked surfaces.
 * Login is unauthenticated; the other three ride on the standard
 * fixture mocks used by the rest of the e2e suite.
 *
 * The runs deliberately use the default axe ruleset (wcag2a + wcag2aa
 * + wcag21a + wcag21aa). A future slice can layer in best-practice
 * rules once the open design backlog items are addressed.
 */

async function expectNoAxeBlockers(page: Page, surface: string) {
  await expectPinnedTheme(page)
  // Callers wait for the surface's real content first; settling the
  // fixture requests as well keeps every project scanning the same DOM.
  await page.waitForLoadState('networkidle')
  const results = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
    .analyze()

  const blocking = results.violations.filter(
    (v) => v.impact === 'critical' || v.impact === 'serious',
  )
  if (blocking.length > 0) {
    // Print every blocker so the CI log shows the offending nodes.
    console.error(JSON.stringify(blocking, null, 2))
  }
  expect(blocking, `${surface} axe blockers (${expectedTheme()})`).toHaveLength(0)
}

test.describe('Issue #766 slice G W4 — WCAG 2.1 AA gate', () => {
  test('login page has no critical or serious axe violations', async ({ page }) => {
    await installApiMocks(page, 'initialized')
    await page.goto('/login')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await expect(page.getByLabel('Username')).toBeVisible()
    await expectNoAxeBlockers(page, 'login')
  })

  test('overview page has no critical or serious axe violations', async ({ page }) => {
    await login(page)
    // login() already waits for the Overview heading; also wait for the
    // late-rendering runtime tile so every project scans the same DOM.
    await expect(page.getByRole('button', { name: 'Restart', exact: true })).toBeVisible()
    await expectNoAxeBlockers(page, 'overview')
  })

  test('configuration page has no critical or serious axe violations', async ({ page }) => {
    await login(page)
    await page.goto('/configuration')
    await expect(page.getByRole('heading', { name: 'Node-RED Configuration', exact: true })).toBeVisible()
    // The raw settings.js editor renders after its fixture loads.
    await expect(page.getByRole('textbox', { name: 'Advanced settings.js' })).toHaveValue(/fixture settings\.js/)
    await expectNoAxeBlockers(page, 'configuration')
  })

  test('security page has no critical or serious axe violations', async ({ page }) => {
    await login(page)
    await page.goto('/security')
    await expect(page.getByRole('heading', { name: 'Node-RED admin auth' })).toBeVisible()
    await expectNoAxeBlockers(page, 'security')
  })

  test('computed muted text colour matches the pinned theme token (#889 S1)', async ({ page }) => {
    await installApiMocks(page, 'initialized')
    await page.goto('/login')
    await expectPinnedTheme(page)

    // L5 token values: dark --ds-text-muted #8e99a8, light #5a6577.
    const expected = expectedTheme() === 'corporateDark' ? 'rgb(142, 153, 168)' : 'rgb(90, 101, 119)'
    const computed = await page.evaluate(() => {
      const probe = document.createElement('p')
      probe.className = 'text-body-secondary'
      probe.textContent = 'probe'
      document.body.appendChild(probe)
      const color = window.getComputedStyle(probe).color
      probe.remove()
      return color
    })
    expect(computed).toBe(expected)
  })

  test('prefers-reduced-motion collapses Tailwind animations to ~0ms', async ({ browser }) => {
    const context = await browser.newContext({ reducedMotion: 'reduce' })
    const page = await context.newPage()
    await login(page)

    // Construct a synthetic element that uses the .animate-spin
    // utility and read its computed animation-duration. The gate in
    // src/index.css collapses every animation to a 0.001ms sentinel
    // when prefers-reduced-motion: reduce is on. Browsers surface
    // this as either '0.001ms' or '1e-06s' depending on the
    // duration unit rounding, so we parse to milliseconds and
    // compare numerically.
    const durationMs = await page.evaluate(() => {
      const probe = document.createElement('div')
      probe.className = 'animate-spin'
      // Attach off-screen so the global stylesheet applies without
      // polluting the rendered tree.
      probe.style.position = 'absolute'
      probe.style.left = '-9999px'
      document.body.appendChild(probe)
      const raw = window.getComputedStyle(probe).animationDuration
      probe.remove()
      const match = raw.trim().match(/^([0-9.eE+-]+)(ms|s)$/)
      if (!match) return Number.NaN
      const value = Number(match[1])
      return match[2] === 's' ? value * 1000 : value
    })

    // The accessibility-safe sentinel is 0.001ms; allow a generous
    // tolerance because some browsers normalise sub-millisecond
    // values to zero or round to 1e-06s.
    expect(durationMs).toBeLessThanOrEqual(1)

    await context.close()
  })

  test('restore confirmation supports mouse and keyboard without bypassing hit testing', async ({ page }) => {
    await login(page)
    await page.goto('/backups')
    await page.locator('button[title="Restore"]').first().click()
    await expect(page.getByText('Restaurar backup')).toBeVisible()
    const confirmation = page.locator('input[placeholder="backup-001"]')
    await confirmation.fill('backup-001', { timeout: 3_000 })
    await expect(confirmation).toHaveValue('backup-001')
    const confirmButton = page.getByRole('button', { name: 'Confirm', exact: true })
    await expect(confirmButton).toBeEnabled()
    await confirmButton.click({ timeout: 3_000 })
    await expect(page.getByText('Restore dry path completed in test mode')).toBeVisible()
  })

  test('restore confirmation can be completed with the keyboard', async ({ page }) => {
    await login(page)
    await page.goto('/backups')
    await page.locator('button[title="Restore"]').first().click()
    await expect(page.getByText('Restaurar backup')).toBeVisible()
    const confirmation = page.locator('input[placeholder="backup-001"]')
    await confirmation.fill('backup-001')
    await confirmation.press('Enter')
    await expect(page.getByText('Restore dry path completed in test mode')).toBeVisible()
  })
})
