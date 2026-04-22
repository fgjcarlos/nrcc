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
      flow: vi.fn(),
    },
  }
})

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
            <Route path="/app/flows/:flowId" element={<FlowsPage flows={flows} loading={false} error={null} operationStatus={{ busy: false }} />} />
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
})
