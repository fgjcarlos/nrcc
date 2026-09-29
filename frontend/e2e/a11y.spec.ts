import { expect, test } from '@playwright/test'
import AxeBuilder from '@axe-core/playwright'
import { installApiMocks, login } from './helpers'

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

test.describe('Issue #766 slice G W4 — WCAG 2.1 AA gate', () => {
  test('login page has no critical or serious axe violations', async ({ page }) => {
    await installApiMocks(page, 'initialized')
    await page.goto('/login')

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
    expect(blocking, 'login axe blockers').toHaveLength(0)
  })

  test('overview page has no critical or serious axe violations', async ({ page }) => {
    await login(page)

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze()

    const blocking = results.violations.filter(
      (v) => v.impact === 'critical' || v.impact === 'serious',
    )
    if (blocking.length > 0) {

      console.error(JSON.stringify(blocking, null, 2))
    }
    expect(blocking, 'overview axe blockers').toHaveLength(0)
  })

  test('configuration page has no critical or serious axe violations', async ({ page }) => {
    await login(page)
    await page.goto('/configuration')

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze()

    const blocking = results.violations.filter(
      (v) => v.impact === 'critical' || v.impact === 'serious',
    )
    if (blocking.length > 0) {

      console.error(JSON.stringify(blocking, null, 2))
    }
    expect(blocking, 'configuration axe blockers').toHaveLength(0)
  })

  test('security page has no critical or serious axe violations', async ({ page }) => {
    await login(page)
    await page.goto('/security')

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze()

    const blocking = results.violations.filter(
      (v) => v.impact === 'critical' || v.impact === 'serious',
    )
    if (blocking.length > 0) {

      console.error(JSON.stringify(blocking, null, 2))
    }
    expect(blocking, 'security axe blockers').toHaveLength(0)
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
      const [value, unit] = raw.split(/^([0-9.eE+-]+)([a-z]*)$/)
      const n = Number(value)
      if (unit === 's') return n * 1000
      return n
    })

    // The accessibility-safe sentinel is 0.001ms; allow a generous
    // tolerance because some browsers normalise sub-millisecond
    // values to zero or round to 1e-06s.
    expect(durationMs).toBeLessThanOrEqual(1)

    await context.close()
  })
})
