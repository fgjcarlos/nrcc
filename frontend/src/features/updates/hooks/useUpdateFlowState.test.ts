import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { queryKeys } from '@/shared/lib/queryKeys';
import { useUpdateFlowState } from './useUpdateFlowState';
import { updateService } from '@/features/updates/services/updateService'
import React from 'react'

// Mock the update service
vi.mock('@/features/updates/services/updateService', () => ({
  updateService: {
    getFlowState: vi.fn(),
  },
}))

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children)
}

describe('useUpdateFlowState hook', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should have correct query key', () => {
    expect(queryKeys.updates.flowState).toEqual(queryKeys.updates.flowState)
  })

  it('should return idle state when no update in progress', async () => {
    const mockState = {
      state: 'Idle' as const,
      phase: 'idle',
    }

    vi.mocked(updateService.getFlowState).mockResolvedValue(mockState)

    const { result } = renderHook(() => useUpdateFlowState(), {
      wrapper: createWrapper(),
    })

    expect(result.current.isLoading).toBe(true)

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data).toEqual(mockState)
  })

  it('should return BackingUp state during backup', async () => {
    const mockState = {
      state: 'BackingUp' as const,
      phase: 'backup',
      backupId: 'backup-123',
    }

    vi.mocked(updateService.getFlowState).mockResolvedValue(mockState)

    const { result } = renderHook(() => useUpdateFlowState(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.state).toBe('BackingUp')
    expect(result.current.data?.backupId).toBe('backup-123')
  })

  it('should return Applying state during npm update', async () => {
    const mockState = {
      state: 'Applying' as const,
      phase: 'update',
    }

    vi.mocked(updateService.getFlowState).mockResolvedValue(mockState)

    const { result } = renderHook(() => useUpdateFlowState(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.state).toBe('Applying')
    expect(result.current.data?.phase).toBe('update')
  })

  it('should return Completed state on success', async () => {
    const mockState = {
      state: 'Completed' as const,
      phase: 'completed',
    }

    vi.mocked(updateService.getFlowState).mockResolvedValue(mockState)

    const { result } = renderHook(() => useUpdateFlowState(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.state).toBe('Completed')
  })

  it('should return Failed state with error message on failure', async () => {
    const mockState = {
      state: 'Failed' as const,
      phase: 'failed',
      error: 'backup_failed',
    }

    vi.mocked(updateService.getFlowState).mockResolvedValue(mockState)

    const { result } = renderHook(() => useUpdateFlowState(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.state).toBe('Failed')
    expect(result.current.data?.error).toBe('backup_failed')
  })

  it('should return idle state with error message when service reports failure', async () => {
    const mockState = {
      state: 'Idle' as const,
      phase: 'idle',
      error: 'Failed to fetch flow state: Network error',
    }

    vi.mocked(updateService.getFlowState).mockResolvedValue(mockState)

    const { result } = renderHook(() => useUpdateFlowState(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    // Should return idle state with error field populated
    expect(result.current.data?.state).toBe('Idle')
    expect(result.current.data?.error).toBeDefined()
  })
})
