import { expect, test } from '@playwright/test'
import { envelope, login } from './helpers'

test('reviews a redacted generation request before explicitly sending and downloading its candidate', async ({ page }) => {
  const flow = [{ id: 'synthetic-node', type: 'inject', name: 'synthetic flow' }]
  const candidate = [{ id: 'candidate-node', type: 'debug', name: 'candidate only' }]
  const requests: Record<string, unknown>[] = []
  let analyzeCalls = 0
  const mutations: string[] = []

  await login(page)
  await page.route('**/api/flows/export', route => route.fulfill({ contentType: 'application/json', body: JSON.stringify(flow) }))
  await page.route('**/api/ai/status', route => route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope({ status: 'ready', provider: 'openai' })) }))
  await page.route('**/api/ai/analyze/flow', async route => {
    const request = route.request().postDataJSON() as Record<string, unknown>
    requests.push(request)
    analyzeCalls += 1
    if (request.previewOnly) {
      await route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope({
        summary: 'Review this redacted request', previewToken: 'synthetic-preview-token',
        request: { endpoint: 'https://provider.invalid', body: { messages: [{ content: '[REDACTED]' }] } },
      })) })
      return
    }
    await route.fulfill({ contentType: 'application/json', body: JSON.stringify(envelope({ summary: 'Candidate ready', candidateFlow: { nodes: candidate } })) })
  })
  page.on('request', (request) => {
    const pathname = new URL(request.url()).pathname
    if (request.method() !== 'GET' && /^\/api\/flows(?:\/|$)/.test(pathname)) mutations.push(`${request.method()} ${pathname}`)
  })

  await page.goto('/flows')
  await expect(page.getByRole('heading', { name: 'Flows' })).toBeVisible()
  await page.getByLabel('Assistant action').selectOption('generate')
  await page.getByLabel('Additional instructions').fill('Generate a candidate')
  await page.getByRole('button', { name: 'Review request' }).click()
  const payload = page.getByText('[REDACTED]')
  await expect(payload).toBeVisible()
  await expect(payload).not.toContainText('synthetic flow')
  expect(analyzeCalls).toBe(1)
  expect(requests[0]).toMatchObject({ action: 'generate', previewOnly: true, flow })
  await expect(page.getByRole('button', { name: 'Send reviewed request' })).toBeVisible()
  expect(analyzeCalls).toBe(1)

  await page.getByRole('button', { name: 'Send reviewed request' }).click()
  await expect(page.getByText('Candidate ready')).toBeVisible()
  await expect(page.getByText('candidate only')).toBeVisible()
  expect(requests[1]).toMatchObject({ action: 'generate', previewToken: 'synthetic-preview-token', flow })
  await expect.poll(() => analyzeCalls).toBe(2)
  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Download nodes JSON' }).click()
  const download = await downloadPromise
  expect(download.suggestedFilename()).toBe('candidate-flow.json')
  const stream = await download.createReadStream()
  let downloaded = ''
  for await (const chunk of stream!) downloaded += chunk.toString()
  expect(JSON.parse(downloaded)).toEqual(candidate)
  expect(mutations).toEqual([])
})
