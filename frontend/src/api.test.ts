import { beforeEach, describe, expect, it, vi } from 'vitest'

type ApiModule = typeof import('./api')

describe('api client', () => {
  let api: ApiModule['api']
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(async () => {
    vi.resetModules()
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    const module = await import('./api')
    api = module.api
  })

  it('stores the csrf token from login and reuses it on later writes', async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          success: true,
          data: {
            user: { id: '1', username: 'admin', role: 'admin', createdAt: '2026-01-01T00:00:00Z' },
            csrfToken: 'csrf-123',
          },
          timestamp: '2026-01-01T00:00:00Z',
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          success: true,
          data: {
            running: true,
            healthy: true,
            pid: 123,
            port: 1880,
            uptimeSec: 42,
            dataDir: '/tmp/node-red',
          },
          timestamp: '2026-01-01T00:00:00Z',
        }),
      })

    await api.login('admin', 'secret')
    await api.runtimeRestart()

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      '/api/auth/login',
      expect.objectContaining({
        credentials: 'include',
        method: 'POST',
      }),
    )

    const loginOptions = fetchMock.mock.calls[0][1] as RequestInit
    expect(loginOptions.body).toBe(JSON.stringify({ username: 'admin', password: 'secret' }))
    expect(new Headers(loginOptions.headers).get('Content-Type')).toBe('application/json')

    const restartOptions = fetchMock.mock.calls[1][1] as RequestInit
    expect(new Headers(restartOptions.headers).get('X-CSRF-Token')).toBe('csrf-123')
  })

  it('clears the csrf token after logout', async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          success: true,
          data: {
            user: { id: '1', username: 'admin', role: 'admin', createdAt: '2026-01-01T00:00:00Z' },
            csrfToken: 'csrf-123',
          },
          timestamp: '2026-01-01T00:00:00Z',
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          success: true,
          data: { loggedOut: true },
          timestamp: '2026-01-01T00:00:00Z',
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          success: true,
          data: {
            running: true,
            healthy: true,
            pid: 123,
            port: 1880,
            uptimeSec: 42,
            dataDir: '/tmp/node-red',
          },
          timestamp: '2026-01-01T00:00:00Z',
        }),
      })

    await api.login('admin', 'secret')
    await api.logout()
    await api.runtimeRestart()

    const restartOptions = fetchMock.mock.calls[2][1] as RequestInit
    expect(new Headers(restartOptions.headers).get('X-CSRF-Token')).toBeNull()
  })

  it('throws APIRequestError with backend details on failed responses', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: async () => ({
        success: false,
        error: {
          code: 'INVALID_CREDENTIALS',
          message: 'Invalid username or password',
        },
        timestamp: '2026-01-01T00:00:00Z',
      }),
    })

    await expect(api.login('admin', 'bad-password')).rejects.toEqual(
      expect.objectContaining({
        name: 'APIRequestError',
        message: 'Invalid username or password',
        status: 401,
        code: 'INVALID_CREDENTIALS',
      }),
    )
  })

  it('serializes diagnostics log filters into the request URL', async () => {
    fetchMock.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: async () => ({
        success: true,
        data: { logs: [], total: 0 },
        timestamp: '2026-01-01T00:00:00Z',
      }),
    })

    await api.diagnosticsLogs({ level: 'error', source: 'runtime', limit: 25, offset: 50 })

    expect(fetchMock.mock.calls[0][0]).toBe('/api/diagnostics/logs?level=error&source=runtime&limit=25&offset=50')
  })

  it('posts flow analysis requests to the selected flow endpoint', async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          success: true,
          data: {
            user: { id: '1', username: 'admin', role: 'admin', createdAt: '2026-01-01T00:00:00Z' },
            csrfToken: 'csrf-123',
          },
          timestamp: '2026-01-01T00:00:00Z',
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => ({
          success: true,
          data: {
            source: { userDir: '/tmp/node-red', flowFile: 'flows.json', path: '/tmp/node-red/flows.json', readOnly: true },
            flow: { id: 'main-flow', label: 'Main Flow', nodeCount: 2, disabledNodeCount: 0, customNodeCount: 0, inboundWireCount: 0, outboundWireCount: 1, subflowUsageCount: 0 },
            advisory: true,
            summary: 'ok',
            strengths: [],
            issues: [],
            suggestions: [],
            provider: { name: 'ollama', model: 'llama3.2', local: true },
          },
          timestamp: '2026-01-01T00:00:00Z',
        }),
      })

    await api.login('admin', 'secret')
    await api.analyzeFlow('main-flow')

    expect(fetchMock.mock.calls[1][0]).toBe('/api/flows/main-flow/analysis')
    const options = fetchMock.mock.calls[1][1] as RequestInit
    expect(options.method).toBe('POST')
    expect(new Headers(options.headers).get('X-CSRF-Token')).toBe('csrf-123')
  })
})
