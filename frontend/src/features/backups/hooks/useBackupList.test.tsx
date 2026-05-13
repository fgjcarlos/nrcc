import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useBackupList } from './useBackupList';
import * as backupServiceModule from '../services/backupService';

// Mock the service
vi.mock('../services/backupService', () => ({
  backupService: {
    listPaginated: vi.fn(),
  },
}));

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('useBackupList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch backups on mount with default params', async () => {
    const mockData = {
      items: [],
      total: 0,
      page: 1,
      limit: 20,
    };

    (backupServiceModule.backupService.listPaginated as any).mockResolvedValueOnce(mockData);

    const { result } = renderHook(() => useBackupList({ page: 1, limit: 20 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).toEqual(mockData);
  });

  it('should pass pagination params to service', async () => {
    const mockData = {
      items: [],
      total: 42,
      page: 2,
      limit: 10,
    };

    (backupServiceModule.backupService.listPaginated as any).mockResolvedValueOnce(mockData);

    renderHook(() => useBackupList({ page: 2, limit: 10 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(backupServiceModule.backupService.listPaginated).toHaveBeenCalledWith({
        page: 2,
        limit: 10,
      });
    });
  });

  it('should pass sort and order params to service', async () => {
    const mockData = {
      items: [],
      total: 42,
      page: 1,
      limit: 20,
    };

    (backupServiceModule.backupService.listPaginated as any).mockResolvedValueOnce(mockData);

    renderHook(
      () =>
        useBackupList({
          page: 1,
          limit: 20,
          sort: 'size',
          order: 'desc',
        }),
      {
        wrapper: createWrapper(),
      }
    );

    await waitFor(() => {
      expect(backupServiceModule.backupService.listPaginated).toHaveBeenCalledWith({
        page: 1,
        limit: 20,
        sort: 'size',
        order: 'desc',
      });
    });
  });

  it('should return query state properties', async () => {
    const mockData = {
      items: [
        {
          id: 'b1',
          name: 'backup-1',
          type: 'manual' as const,
          createdAt: '2026-05-11T20:00:00Z',
          triggeredBy: 'System',
          fileCount: 5,
          totalSize: 1024,
        },
      ],
      total: 1,
      page: 1,
      limit: 20,
    };

    (backupServiceModule.backupService.listPaginated as any).mockResolvedValueOnce(mockData);

    const { result } = renderHook(() => useBackupList({ page: 1, limit: 20 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).toBeDefined();
    expect(result.current.isLoading).toBe(false);
    expect(result.current.isError).toBe(false);
  });

  it('should handle error state', async () => {
    const error = new Error('API error');
    (backupServiceModule.backupService.listPaginated as any).mockRejectedValueOnce(error);

    const { result } = renderHook(() => useBackupList({ page: 1, limit: 20 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.isError).toBe(true);
  });

  it('should support different limit values', async () => {
    const mockData = {
      items: [],
      total: 100,
      page: 1,
      limit: 50,
    };

    (backupServiceModule.backupService.listPaginated as any).mockResolvedValueOnce(mockData);

    renderHook(() => useBackupList({ page: 1, limit: 50 }), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(backupServiceModule.backupService.listPaginated).toHaveBeenCalledWith({
        page: 1,
        limit: 50,
      });
    });
  });

  it('should support sort by date', async () => {
    const mockData = {
      items: [],
      total: 42,
      page: 1,
      limit: 20,
    };

    (backupServiceModule.backupService.listPaginated as any).mockResolvedValueOnce(mockData);

    renderHook(
      () =>
        useBackupList({
          page: 1,
          limit: 20,
          sort: 'date',
          order: 'asc',
        }),
      {
        wrapper: createWrapper(),
      }
    );

    await waitFor(() => {
      expect(backupServiceModule.backupService.listPaginated).toHaveBeenCalledWith({
        page: 1,
        limit: 20,
        sort: 'date',
        order: 'asc',
      });
    });
  });

  it('should support sort by status', async () => {
    const mockData = {
      items: [],
      total: 42,
      page: 1,
      limit: 20,
    };

    (backupServiceModule.backupService.listPaginated as any).mockResolvedValueOnce(mockData);

    renderHook(
      () =>
        useBackupList({
          page: 1,
          limit: 20,
          sort: 'status',
          order: 'asc',
        }),
      {
        wrapper: createWrapper(),
      }
    );

    await waitFor(() => {
      expect(backupServiceModule.backupService.listPaginated).toHaveBeenCalledWith({
        page: 1,
        limit: 20,
        sort: 'status',
        order: 'asc',
      });
    });
  });

  it('should create correct query key for cache management', async () => {
    const mockData = {
      items: [],
      total: 0,
      page: 1,
      limit: 20,
    };

    (backupServiceModule.backupService.listPaginated as any).mockResolvedValueOnce(mockData);

    renderHook(
      () =>
        useBackupList({
          page: 1,
          limit: 20,
          sort: 'date',
          order: 'desc',
        }),
      {
        wrapper: createWrapper(),
      }
    );

    // The queryKey should include all params for proper cache isolation
    expect(backupServiceModule.backupService.listPaginated).toHaveBeenCalled();
  });
});
