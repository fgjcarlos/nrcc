import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LogsView } from './LogsView'
import * as logsHooks from '../hooks'
import type { LogLevel } from '@/shared/types'

vi.mock('../hooks', () => ({
  useLogsData: vi.fn(),
  useLogsActions: vi.fn(),
}))

const mockSetLevelFilter = vi.fn()
const mockSetIsPaused = vi.fn()
const mockRefetch = vi.fn()
const mockHandleClear = vi.fn()
const mockHandleDownload = vi.fn()
const mockGetLevelColor = vi.fn((_level: LogLevel) => 'text-blue-500' as const)

describe('LogsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(logsHooks.useLogsActions).mockReturnValue({
      handleClear: mockHandleClear,
      handleDownload: mockHandleDownload,
      getLevelColor: mockGetLevelColor,
    })
  })

  it('shows a loading state while logs are loading', () => {
    vi.mocked(logsHooks.useLogsData).mockReturnValue({
      logs: [],
      isLoading: true,
      isError: false,
      error: null,
      levelFilter: ['info', 'warn', 'error'],
      setLevelFilter: mockSetLevelFilter,
      isPaused: false,
      setIsPaused: mockSetIsPaused,
      refetch: mockRefetch,
    })

    render(<LogsView />)

    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('shows an error state when log fetching fails', () => {
    vi.mocked(logsHooks.useLogsData).mockReturnValue({
      logs: [],
      isLoading: false,
      isError: true,
      error: new Error('Failed to fetch logs'),
      levelFilter: ['info', 'warn', 'error'],
      setLevelFilter: mockSetLevelFilter,
      isPaused: false,
      setIsPaused: mockSetIsPaused,
      refetch: mockRefetch,
    })

    render(<LogsView />)

    expect(screen.getByText('An error occurred')).toBeInTheDocument()
  })

  it('shows an empty state when there are no logs', () => {
    vi.mocked(logsHooks.useLogsData).mockReturnValue({
      logs: [],
      isLoading: false,
      isError: false,
      error: null,
      levelFilter: ['info', 'warn', 'error'],
      setLevelFilter: mockSetLevelFilter,
      isPaused: false,
      setIsPaused: mockSetIsPaused,
      refetch: mockRefetch,
    })

    render(<LogsView />)

    expect(screen.getByText('No logs available')).toBeInTheDocument()
  })

  it('renders logs and wires toolbar actions', async () => {
    const user = userEvent.setup()

    vi.mocked(logsHooks.useLogsData).mockReturnValue({
      logs: [
        {
          id: 'log-1',
          timestamp: '2026-05-22T12:00:00Z',
          level: 'info',
          message: 'Node-RED started',
        },
      ],
      isLoading: false,
      isError: false,
      error: null,
      levelFilter: ['info', 'warn', 'error'],
      setLevelFilter: mockSetLevelFilter,
      isPaused: false,
      setIsPaused: mockSetIsPaused,
      refetch: mockRefetch,
    })

    render(<LogsView />)

    expect(screen.getByRole('heading', { name: 'Logs' })).toBeInTheDocument()
    expect(screen.getByText('Node-RED started')).toBeInTheDocument()

    const buttons = screen.getAllByRole('button')
    await user.click(buttons[0])
    await user.click(buttons[1])
    await user.click(buttons[2])

    expect(mockSetIsPaused).toHaveBeenCalledWith(true)
    expect(mockHandleClear).toHaveBeenCalledWith(mockRefetch)
    expect(mockHandleDownload).toHaveBeenCalledWith([
      {
        id: 'log-1',
        timestamp: '2026-05-22T12:00:00Z',
        level: 'info',
        message: 'Node-RED started',
      },
    ])
  })
})
