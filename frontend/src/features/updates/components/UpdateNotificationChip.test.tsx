import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { UpdateNotificationChip } from '@/features/updates/components/UpdateNotificationChip'
import * as useUpdateStatusHook from '@/features/updates/hooks'
import * as useUpdateFlowStateHook from '@/features/updates/hooks'
import React from 'react'

// Mock dependencies
vi.mock('@/hooks/useUpdateStatus')
vi.mock('@/features/updates/hooks')
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => vi.fn(),
  }
})

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(
      BrowserRouter,
      {},
      React.createElement(QueryClientProvider, { client: queryClient }, children)
    )
}

describe('UpdateNotificationChip component', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    // Default mock for flow state: Idle (no active update)
    vi.spyOn(useUpdateFlowStateHook, 'useUpdateFlowState').mockReturnValue({
      data: { state: 'Idle' },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)
  })

  it('should render null when no update is available', () => {
    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: {
        currentVersion: '3.0.0',
        latestVersion: '3.0.0',
        updateAvailable: false,
        checkedAt: new Date().toISOString(),
      },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    const { container } = render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    expect(container.firstChild).toBeNull()
  })

  it('should render chip when update is available', () => {
    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: {
        currentVersion: '3.0.0',
        latestVersion: '3.1.0',
        updateAvailable: true,
        checkedAt: new Date().toISOString(),
      },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    expect(screen.getByText('Update available')).toBeInTheDocument()
  })

  it('should return null when loading', () => {
    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      isSuccess: false,
      status: 'pending',
    } as any)

    const { container } = render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    expect(container.firstChild).toBeNull()
  })

  it('should hide chip when dismissed', () => {
    const status = {
      currentVersion: '3.0.0',
      latestVersion: '3.1.0',
      updateAvailable: true,
      checkedAt: new Date().toISOString(),
    }

    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: status,
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    // Set dismissed version
    localStorage.setItem('cc-update-dismissed-version', '3.1.0')

    const { container } = render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    expect(container.firstChild).toBeNull()
  })

  it('should show again when version changes after dismissal', async () => {
    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: {
        currentVersion: '3.0.0',
        latestVersion: '3.2.0', // Different version
        updateAvailable: true,
        checkedAt: new Date().toISOString(),
      },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    // Dismissed version is different
    localStorage.setItem('cc-update-dismissed-version', '3.1.0')

    render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    // Wait for useEffect to set dismissed state
    await waitFor(() => {
      expect(screen.getByText('Update available')).toBeInTheDocument()
    })
  })

  it('should have proper accessibility attributes', () => {
    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: {
        currentVersion: '3.0.0',
        latestVersion: '3.1.0',
        updateAvailable: true,
        checkedAt: new Date().toISOString(),
      },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    const mainButton = screen.getByLabelText('Update available, click to go to updates page')
    const dismissButton = screen.getByLabelText('Dismiss update notification')

    expect(mainButton).toBeInTheDocument()
    expect(dismissButton).toBeInTheDocument()
  })

  it('should show spinner when update is active (BackingUp state)', () => {
    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: {
        currentVersion: '3.0.0',
        latestVersion: '3.1.0',
        updateAvailable: true,
        checkedAt: new Date().toISOString(),
      },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    vi.spyOn(useUpdateFlowStateHook, 'useUpdateFlowState').mockReturnValue({
      data: { state: 'BackingUp', phase: 'backup' },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    expect(screen.getByText('Backing up...')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveAttribute('aria-live', 'polite')
  })

  it('should show updating text when in Applying state', () => {
    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: {
        currentVersion: '3.0.0',
        latestVersion: '3.1.0',
        updateAvailable: true,
        checkedAt: new Date().toISOString(),
      },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    vi.spyOn(useUpdateFlowStateHook, 'useUpdateFlowState').mockReturnValue({
      data: { state: 'Applying', phase: 'apply' },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    expect(screen.getByText('Updating...')).toBeInTheDocument()
  })

  it('should show checkmark when update is completed', () => {
    vi.spyOn(useUpdateStatusHook, 'useUpdateStatus').mockReturnValue({
      data: {
        currentVersion: '3.1.0', // Version updated
        latestVersion: '3.1.0',
        updateAvailable: false,
        checkedAt: new Date().toISOString(),
      },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    vi.spyOn(useUpdateFlowStateHook, 'useUpdateFlowState').mockReturnValue({
      data: { state: 'Completed' },
      isLoading: false,
      isError: false,
      isSuccess: true,
      status: 'success',
    } as any)

    render(<UpdateNotificationChip />, {
      wrapper: createWrapper(),
    })

    expect(screen.getByText('Updated')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Update completed')
  })
})
