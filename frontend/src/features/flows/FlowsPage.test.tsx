import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { api, type FlowList } from '../../api'
import { createTestQueryClient } from '../../test/utils'
import { FlowsPage } from './FlowsPage'

vi.mock('../../api', async () => {
  const actual = await vi.importActual<typeof import('../../api')>('../../api')

  return {
    ...actual,
    api: {
      ...actual.api,
      flows: vi.fn(),
      operationsStatus: vi.fn(),
      flow: vi.fn(),
      analyzeFlow: vi.fn(),
    },
  }
})

vi.mock('../auth/useAuth', () => ({
  useAuth: vi.fn(() => ({
    user: { id: 'test-user', role: 'admin', username: 'test' },
    login: vi.fn(),
    logout: vi.fn(),
  })),
}))

const flows: FlowList = {
  source: {
    userDir: '/var/lib/node-red',
    flowFile: 'flows.json',
    path: '/var/lib/node-red/flows.json',
    readOnly: true,
    updatedAt: '2026-01-01T00:00:00Z',
  },
  summary: {
    flowCount: 1,
    nodeCount: 2,
    disabledNodeCount: 1,
    customNodeCount: 1,
    inboundWireCount: 1,
    outboundWireCount: 1,
    subflowUsageCount: 0,
  },
  items: [
    {
      id: 'main-flow',
      label: 'Main Flow',
      nodeCount: 2,
      disabledNodeCount: 1,
      customNodeCount: 1,
      inboundWireCount: 1,
      outboundWireCount: 1,
      subflowUsageCount: 0,
    },
  ],
}

describe('FlowsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.flows).mockResolvedValue(flows)
    vi.mocked(api.operationsStatus).mockResolvedValue({ busy: false })
  })

  it('loads detail for the selected flow route', async () => {
    vi.mocked(api.flow).mockResolvedValue({
      source: flows.source,
      flow: {
        ...flows.items[0],
        nodeTypes: [{ type: 'acme-widget', count: 1, custom: true }],
        nodes: [{ id: 'n1', type: 'acme-widget', name: 'Widget', disabled: true, wireCount: 1 }],
      },
    })

    render(
      <QueryClientProvider client={createTestQueryClient()}>
        <MemoryRouter initialEntries={['/app/flows/main-flow']}>
          <Routes>
            <Route path="/app/flows/:flowId" element={<FlowsPage />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(screen.getByRole('heading', { name: 'Flows' })).toBeInTheDocument()
    expect(screen.getByText('Main Flow')).toBeInTheDocument()

    await waitFor(() => {
      expect(api.flow).toHaveBeenCalledWith('main-flow')
    })

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('custom')).toBeInTheDocument()
  })

  it('runs advisory analysis for the selected flow', async () => {
    vi.mocked(api.flow).mockResolvedValue({
      source: flows.source,
      flow: {
        ...flows.items[0],
        nodeTypes: [{ type: 'inject', count: 1, custom: false }],
        nodes: [{ id: 'n1', type: 'inject', name: 'Start', disabled: false, wireCount: 1 }],
      },
    })
    vi.mocked(api.analyzeFlow).mockResolvedValue({
      source: flows.source,
      flow: flows.items[0],
      advisory: true,
      summary: 'Resumen claro del flujo principal.',
      strengths: ['Tiene una entrada simple y visible.'],
      issues: ['Solo existe un punto de observacion.'],
      suggestions: ['Agregar mas validacion operacional.'],
      provider: { name: 'ollama', model: 'llama3.2', local: true },
    })

    render(
      <QueryClientProvider client={createTestQueryClient()}>
        <MemoryRouter initialEntries={['/app/flows/main-flow']}>
          <Routes>
            <Route path="/app/flows/:flowId" element={<FlowsPage />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(await screen.findByRole('button', { name: 'Analyze selected flow' })).toBeInTheDocument()
    screen.getByRole('button', { name: 'Analyze selected flow' }).click()

    await waitFor(() => {
      expect(api.analyzeFlow).toHaveBeenCalledWith('main-flow')
    })

    expect(await screen.findByText('Resumen claro del flujo principal.')).toBeInTheDocument()
    expect(screen.getByText('Tiene una entrada simple y visible.')).toBeInTheDocument()
    expect(screen.getByText('ollama')).toBeInTheDocument()
  })
})
