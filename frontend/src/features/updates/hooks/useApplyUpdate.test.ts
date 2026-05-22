import { describe, it, expect, beforeEach, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useApplyUpdate } from '@/features/updates/hooks/useApplyUpdate'
import { updateService } from '@/features/updates/services/updateService'
import React from 'react'

// Mock the update service
vi.mock('@/features/updates/services/updateService', () => ({
  updateService: {
    applyUpdate: vi.fn(),
  },
}))

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children)
}

describe('useApplyUpdate hook', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should call updateService.applyUpdate when mutate is called', async () => {
    const mockResponse = {
      success: true,
      message: 'Update apply initiated',
    }

    vi.mocked(updateService.applyUpdate).mockResolvedValue(mockResponse)

    const { result } = renderHook(() => useApplyUpdate(), {
      wrapper: createWrapper(),
    })

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(vi.mocked(updateService.applyUpdate)).toHaveBeenCalledTimes(1)
    expect(result.current.data).toEqual(mockResponse)
  })

  it('should handle successful apply response', async () => {
    const mockResponse = {
      success: true,
      message: 'Update applying',
      fromVersion: '3.0.0',
      toVersion: '3.1.0',
    }

    vi.mocked(updateService.applyUpdate).mockResolvedValue(mockResponse)

    const { result } = renderHook(() => useApplyUpdate(), {
      wrapper: createWrapper(),
    })

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.success).toBe(true)
    expect(result.current.data?.fromVersion).toBe('3.0.0')
  })

  it('should handle apply error response', async () => {
    const mockResponse = {
      success: false,
      message: 'Update already in progress',
    }

    vi.mocked(updateService.applyUpdate).mockResolvedValue(mockResponse)

    const { result } = renderHook(() => useApplyUpdate(), {
      wrapper: createWrapper(),
    })

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    // Note: useMutation considers this a success (promise resolved)
    // even though the response indicates failure
    expect(result.current.data?.success).toBe(false)
  })

  it('should handle network error', async () => {
    vi.mocked(updateService.applyUpdate).mockRejectedValue(
      new Error('Network error')
    )

    const { result } = renderHook(() => useApplyUpdate(), {
      wrapper: createWrapper(),
    })

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })

    expect(result.current.error).toBeDefined()
  })

  it('should expose mutate function for triggering update', async () => {
    const mockResponse = {
      success: true,
      message: 'Update initiated',
    }

    vi.mocked(updateService.applyUpdate).mockResolvedValue(mockResponse)

    const { result } = renderHook(() => useApplyUpdate(), {
      wrapper: createWrapper(),
    })

    // Should have mutate function
    expect(typeof result.current.mutate).toBe('function')
    expect(typeof result.current.mutateAsync).toBe('function')

    result.current.mutate()

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })
  })
})
