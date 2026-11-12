import { expect, test, type Page } from '@playwright/test'

/**
 * Shared accessibility helpers — issue #889.
 *
 * Every Playwright project pins a theme (see playwright.config.ts) and
 * declares it in `metadata.theme`; the specs prove that the pinned
 * `data-theme` is the one rendered before asserting anything else.
 */
export function expectedTheme(): string {
  const theme = test.info().project.metadata?.theme
  if (typeof theme !== 'string') throw new Error(`project ${test.info().project.name} has no metadata.theme`)
  return theme
}

export function isMobileProject(): boolean {
  return test.info().project.metadata?.mobile === true
}

export async function expectPinnedTheme(page: Page) {
  await expect
    .poll(() => page.evaluate(() => document.documentElement.dataset.theme))
    .toBe(expectedTheme())
}

export interface FocusStop {
  /** Short description: tag, id/name and accessible text. */
  label: string
  /** Region of the app the element belongs to. */
  region: 'skip-link' | 'header' | 'sidebar' | 'main' | 'dialog' | 'other'
  /** True when the focused element (or the control that represents it) shows a focus indicator. */
  visible: boolean
  detail: string
}

/**
 * Records the unfocused outline/box-shadow of every focusable element so a
 * focus indicator is only credited when focus actually changes the
 * rendering (e.g. the active locale button carries a ring at rest).
 */
export async function snapshotFocusBaseline(page: Page) {
  await page.evaluate(() => {
    const active = document.activeElement as HTMLElement | null
    active?.blur()
    const baseline = new WeakMap<Element, { boxShadow: string; outline: string }>()
    const selector = 'a[href], button, input, select, textarea, summary, [tabindex]'
    document.querySelectorAll(selector).forEach((el) => {
      const cs = window.getComputedStyle(el)
      baseline.set(el, {
        boxShadow: cs.boxShadow,
        outline: `${cs.outlineStyle} ${cs.outlineWidth} ${cs.outlineColor}`,
      })
    })
    ;(window as unknown as { __focusBaseline: typeof baseline }).__focusBaseline = baseline
  })
}

/**
 * Presses Tab once and describes the newly focused element. Visible means
 * a rendered (non-transparent, non-zero-size) element with a
 * non-transparent outline of width > 0, or a box-shadow that differs
 * from the element's unfocused box-shadow. The daisyUI drawer toggle is a
 * transparent checkbox; its indicator is drawn on the header label
 * (`label.drawer-button`), so that label is inspected instead.
 */
export async function tabAndProbe(page: Page): Promise<FocusStop | null> {
  await page.keyboard.press('Tab')
  return page.evaluate((): FocusStop | null => {
    const active = document.activeElement as HTMLElement | null
    if (!active || active === document.body) return null

    let probe: Element = active
    if (active.classList.contains('drawer-toggle') && active.id) {
      probe = document.querySelector(`label.drawer-button[for="${active.id}"]`) ?? active
    }

    const baselineMap = (window as unknown as {
      __focusBaseline?: WeakMap<Element, { boxShadow: string; outline: string }>
    }).__focusBaseline
    const cs = window.getComputedStyle(probe)
    const alpha = (color: string) => {
      const m = color.match(/rgba?\(([^)]+)\)/)
      if (!m) return 1
      const parts = m[1].split(/[\s,/]+/).filter(Boolean)
      return parts.length >= 4 ? Number(parts[3]) : 1
    }
    // A transparent or zero-sized control cannot show its own indicator.
    const rect = probe.getBoundingClientRect()
    const rendered = Number(cs.opacity) > 0 && rect.width > 0 && rect.height > 0
    const outlineVisible =
      rendered &&
      cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) > 0 && alpha(cs.outlineColor) > 0
    const baseline = baselineMap?.get(probe)
    const shadowVisible =
      rendered &&
      cs.boxShadow !== 'none' && (!baseline || baseline.boxShadow !== cs.boxShadow)
    const outlineChanged =
      !baseline || baseline.outline !== `${cs.outlineStyle} ${cs.outlineWidth} ${cs.outlineColor}`

    const region = active.closest('[data-testid="app-topbar"]')
      ? 'header'
      : active.closest('[data-testid="app-sidebar"]')
        ? 'sidebar'
        : active.closest('[role="dialog"]')
          ? 'dialog'
          : active.matches('a[href="#main-content"]')
            ? 'skip-link'
            : active.classList.contains('drawer-toggle')
              ? 'header'
              : active.closest('main')
                ? 'main'
                : 'other'

    const name =
      active.getAttribute('aria-label') ||
      active.id ||
      (active.textContent ?? '').trim().slice(0, 40) ||
      active.getAttribute('name') ||
      ''
    return {
      label: `${active.tagName.toLowerCase()}${name ? ` "${name}"` : ''}`,
      region,
      visible: (outlineVisible && outlineChanged) || shadowVisible,
      detail: `outline=${cs.outlineStyle} ${cs.outlineWidth} ${cs.outlineColor}; box-shadow=${cs.boxShadow}`,
    }
  })
}
