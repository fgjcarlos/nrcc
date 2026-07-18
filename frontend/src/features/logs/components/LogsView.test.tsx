import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { LogsView } from './LogsView'
import * as logsHooks from '../hooks'
import type { LogEntry, LogLevel } from '@/shared/types'

vi.mock('../hooks', () => ({
  useLogsData: vi.fn(),
  useLogsActions: vi.fn(),
}))

const mockSetLevelFilter = vi.fn()
const mockSetIsPaused = vi.fn()
const mockRefetch = vi.fn()
const mockHandleClear = vi.fn()
const mockHandleCopy = vi.fn()
const mockHandleDownload = vi.fn()
const mockHandleDownloadJSON = vi.fn()
const mockGetLevelColor = vi.fn((_level: LogLevel) => 'text-blue-500' as const)
const mockToggleLevel = vi.fn(
  (current: LogLevel[], level: LogLevel) =>
    current.includes(level) ? current.filter(l => l !== level) : [...current, level],
)

const mockData = (overrides: Partial<ReturnType<typeof logsHooks.useLogsData>> = {}) => {
  vi.mocked(logsHooks.useLogsData).mockReturnValue({
    logs: [],
    isLoading: false,
    isError: false,
    error: null,
    levelFilter: ['debug', 'info', 'warn', 'error'],
    setLevelFilter: mockSetLevelFilter,
    isPaused: false,
    setIsPaused: mockSetIsPaused,
    refetch: mockRefetch,
    ...overrides,
  })
}

describe('LogsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(logsHooks.useLogsActions).mockReturnValue({
      handleClear: mockHandleClear,
      handleCopy: mockHandleCopy,
      handleDownload: mockHandleDownload,
      handleDownloadJSON: mockHandleDownloadJSON,
      toggleLevel: mockToggleLevel,
      getLevelColor: mockGetLevelColor,
    })
  })

  it('shows a loading state while logs are loading', () => {
    mockData({ isLoading: true })
    render(<LogsView />)
    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('shows an error state when log fetching fails', () => {
    mockData({ isError: true, error: new Error('Failed to fetch logs') })
    render(<LogsView />)
    expect(screen.getByText('An error occurred')).toBeInTheDocument()
  })

  it('shows an empty state when there are no logs', () => {
    mockData()
    render(<LogsView />)
    expect(screen.getByText('No logs available')).toBeInTheDocument()
  })

  it('defaults the level filter to every level including debug (#461)', () => {
    mockData()
    render(<LogsView />)
    // 4 chips with role=switch, all aria-checked=true
    const chips = screen.getAllByRole('switch')
    expect(chips).toHaveLength(4)
    chips.forEach(chip => expect(chip).toHaveAttribute('aria-checked', 'true'))
  })

  it('renders logs and wires toolbar actions (existing contract)', async () => {
    const user = userEvent.setup()
    const log: LogEntry = {
      id: 'log-1',
      timestamp: '2026-05-22T12:00:00Z',
      level: 'info',
      message: 'Node-RED started',
    }
    mockData({ logs: [log] })

    render(<LogsView />)

    expect(screen.getByRole('heading', { name: 'Logs' })).toBeInTheDocument()
    expect(screen.getByText('Node-RED started')).toBeInTheDocument()

    // First three action-bar buttons: Pause, Clear, Download (preserved order).
    const buttons = screen.getAllByRole('button').filter(b => b.querySelector('svg'))
    await user.click(buttons[0])
    await user.click(buttons[1])
    await user.click(buttons[2])

    expect(mockSetIsPaused).toHaveBeenCalledWith(true)
    expect(mockHandleClear).toHaveBeenCalledWith(mockRefetch)
    expect(mockHandleDownload).toHaveBeenCalledWith([log])
  })

  it('wires the new Copy / JSON / Refresh buttons after the original three', async () => {
    const user = userEvent.setup()
    const log: LogEntry = {
      id: 'log-1',
      timestamp: '2026-05-22T12:00:00Z',
      level: 'info',
      message: 'Node-RED started',
    }
    mockData({ logs: [log] })

    render(<LogsView />)

    const buttons = screen.getAllByRole('button').filter(b => b.querySelector('svg'))
    // Pause, Clear, Download .txt, Download .json, Copy, Refresh
    await user.click(buttons[3])
    await user.click(buttons[4])
    await user.click(buttons[5])

    expect(mockHandleDownloadJSON).toHaveBeenCalledWith([log])
    expect(mockHandleCopy).toHaveBeenCalledWith([log])
    expect(mockRefetch).toHaveBeenCalled()
  })

  it('toggles level chips through the hook helper', async () => {
    const user = userEvent.setup()
    mockData({ levelFilter: ['info'] })

    render(<LogsView />)

    const infoChip = screen.getByRole('switch', { name: /toggle info level/i })
    expect(infoChip).toHaveAttribute('aria-checked', 'true')

    await user.click(infoChip)

    expect(mockToggleLevel).toHaveBeenCalledWith(['info'], 'info')
    expect(mockSetLevelFilter).toHaveBeenCalled()
  })
})
