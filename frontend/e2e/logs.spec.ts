import { expect, test } from '@playwright/test'
import { envelope, login } from './helpers'

const firstStart = '2026-04-01T12:00:00Z'
const nextStart = '2026-04-01T12:05:00Z'
const firstEntries = [
  { id: 'g:1', timestamp: '2026-04-01T12:01:00Z', level: 'error', source: 'runtime', message: 'first launch error', runStartedAt: firstStart },
  { id: 'g:2', timestamp: '2026-04-01T12:02:00Z', level: 'info', source: 'runtime', message: 'first launch info', runStartedAt: firstStart },
]

function historyPage(startedAt: string, entries: typeof firstEntries) {
  return envelope({ entries, generation: 'generation-1', highWatermark: 'generation-1:2', startedAt })
}

test.describe('runtime logs', () => {
  test('selects chart buckets and detects a runtime restart on SSE reconnect before pausing', async ({ page }) => {
    await login(page)
    let runtimeRestarted = false
    let historyCalls = 0
    let streamCalls = 0
    const authHeaders: string[] = []

    await page.route(/\/api\/logs(?:\?.*)?$/, async (route) => {
      historyCalls += 1
      const url = new URL(route.request().url())
      const sinceRestart = url.searchParams.get('sinceRestart') === 'true'
      const entries = runtimeRestarted
        ? [{ id: 'g2:1', timestamp: '2026-04-01T12:05:01Z', level: 'error', source: 'runtime', message: 'second launch error', runStartedAt: nextStart }]
        : sinceRestart
          ? firstEntries
          : firstEntries
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(historyPage(runtimeRestarted ? nextStart : firstStart, entries)) })
    })
    await page.route(/\/api\/logs\/stream(?:\?.*)?$/, async (route) => {
      streamCalls += 1
      authHeaders.push(route.request().headers().authorization ?? '')
      runtimeRestarted = streamCalls >= 2
      const announcedStart = runtimeRestarted ? nextStart : firstStart
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        headers: { 'cache-control': 'no-cache' },
        body: `event: connected\ndata: ${JSON.stringify({ generation: 'generation-1', highWatermark: 'generation-1:2', startedAt: announcedStart })}\n\n`,
      })
    })

    await page.goto('/maintenance/logs')
    await expect(page.getByRole('heading', { name: 'Runtime logs' })).toBeVisible()
    const chart = page.getByRole('group', { name: 'Events per level over time' })
    await chart.getByRole('button').first().click()
    const table = page.getByRole('table', { name: 'Runtime log entries' })
    await expect(table.getByText('first launch error')).toBeVisible()
    await expect(table.getByText('first launch info')).toHaveCount(0)
    await page.getByRole('button', { name: 'Clear chart selection' }).click()
    await expect(table.getByText('first launch info')).toBeVisible()

    await page.getByRole('button', { name: 'Since last restart' }).click()
    await expect(table.getByText('first launch error')).toBeVisible()
    await page.getByRole('button', { name: 'Resume live tail' }).click()
    await expect(page.getByRole('button', { name: 'Pause live tail' })).toHaveAttribute('aria-pressed', 'true')
    await expect(table.getByText('second launch error')).toBeVisible({ timeout: 15_000 })
    await expect(table.getByText('first launch error')).toHaveCount(0)
    expect(streamCalls).toBeGreaterThanOrEqual(3)
    expect(authHeaders.length).toBe(streamCalls)
    expect(authHeaders.every((header) => header.startsWith('Bearer '))).toBe(true)

    await page.getByRole('button', { name: 'Pause live tail' }).click()
    await expect(page.getByRole('button', { name: 'Resume live tail' })).toHaveAttribute('aria-pressed', 'false')
    const pausedStreamCalls = streamCalls
    await page.waitForTimeout(1_200)
    expect(streamCalls).toBe(pausedStreamCalls)
    expect(historyCalls).toBeGreaterThanOrEqual(4)
  })

  test('a history failure shows an error instead of claiming there are no matching logs', async ({ page }) => {
    await login(page)
    await page.route(/\/api\/logs(?:\?.*)?$/, async (route) => {
      await route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ success: false, error: { message: 'unavailable' } }) })
    })
    await page.goto('/maintenance/logs')
    await expect(page.getByRole('alert')).toContainText('Unable to load runtime logs')
    await expect(page.getByText('No logs match these filters.')).toHaveCount(0)
  })
})
