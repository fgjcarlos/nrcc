import { describe, it, expect, vi, beforeEach } from 'vitest';
import { backupService } from './backupService';
import * as api from '@/shared/lib';

// Mock the api barrel used by backupService
vi.mock('@/shared/lib', () => ({
  default: {
    get: vi.fn(),
  },
}));

describe('backupService.listPaginated', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should call GET /backups with pagination params', async () => {
    const mockResponse = {
      data: {
        items: [],
        total: 0,
        page: 1,
        limit: 20,
      },
    };

    (api.default.get as any).mockResolvedValueOnce(mockResponse);

    await backupService.listPaginated({ page: 1, limit: 20 });

    const callArg = (api.default.get as any).mock.calls[0][0];
    expect(callArg).toContain('page=1');
    expect(callArg).toContain('limit=20');
  });

  it('should include sort and order params when provided', async () => {
    const mockResponse = {
      data: {
        items: [],
        total: 0,
        page: 1,
        limit: 20,
      },
    };

    (api.default.get as any).mockResolvedValueOnce(mockResponse);

    await backupService.listPaginated({
      page: 1,
      limit: 20,
      sort: 'size',
      order: 'desc',
    });

    const callArg = (api.default.get as any).mock.calls[0][0];
    expect(callArg).toContain('sort=size');
    expect(callArg).toContain('order=desc');
  });

  it('should return paginated response with normalized items', async () => {
    const mockResponse = {
      data: {
        items: [
          {
            id: 'backup-1',
            name: 'test-backup',
            type: 'manual',
            createdAt: '2026-05-11T20:00:00Z',
            fileCount: 5,
            sizeBytes: 1024,
          },
        ],
        total: 42,
        page: 1,
        limit: 20,
      },
    };

    (api.default.get as any).mockResolvedValueOnce(mockResponse);

    const result = await backupService.listPaginated({ page: 1, limit: 20 });

    expect(result.items).toHaveLength(1);
    expect(result.items[0].id).toBe('backup-1');
    expect(result.total).toBe(42);
    expect(result.page).toBe(1);
    expect(result.limit).toBe(20);
  });

  it('should handle second page request correctly', async () => {
    const mockResponse = {
      data: {
        items: [
          {
            id: 'backup-21',
            name: 'backup-21',
            type: 'auto',
            createdAt: '2026-05-10T20:00:00Z',
            fileCount: 3,
            sizeBytes: 2048,
          },
          {
            id: 'backup-22',
            name: 'backup-22',
            type: 'auto',
            createdAt: '2026-05-09T20:00:00Z',
            fileCount: 2,
            sizeBytes: 512,
          },
        ],
        total: 42,
        page: 2,
        limit: 20,
      },
    };

    (api.default.get as any).mockResolvedValueOnce(mockResponse);

    const result = await backupService.listPaginated({ page: 2, limit: 20 });

    expect(result.items).toHaveLength(2);
    expect(result.page).toBe(2);
    expect(result.total).toBe(42);
    expect(result.items[0].id).toBe('backup-21');
    expect(result.items[1].id).toBe('backup-22');
  });

  it('should handle empty items list', async () => {
    const mockResponse = {
      data: {
        items: [],
        total: 20,
        page: 2,
        limit: 20,
      },
    };

    (api.default.get as any).mockResolvedValueOnce(mockResponse);

    const result = await backupService.listPaginated({ page: 2, limit: 20 });

    expect(result.items).toHaveLength(0);
    expect(result.total).toBe(20);
  });

  it('should support sort by date ascending', async () => {
    const mockResponse = {
      data: {
        items: [],
        total: 0,
        page: 1,
        limit: 20,
      },
    };

    (api.default.get as any).mockResolvedValueOnce(mockResponse);

    await backupService.listPaginated({
      page: 1,
      limit: 20,
      sort: 'date',
      order: 'asc',
    });

    const callArg = (api.default.get as any).mock.calls[0][0];
    expect(callArg).toContain('sort=date');
    expect(callArg).toContain('order=asc');
  });

  it('should support sort by status', async () => {
    const mockResponse = {
      data: {
        items: [],
        total: 0,
        page: 1,
        limit: 20,
      },
    };

    (api.default.get as any).mockResolvedValueOnce(mockResponse);

    await backupService.listPaginated({
      page: 1,
      limit: 20,
      sort: 'status',
      order: 'asc',
    });

    const callArg = (api.default.get as any).mock.calls[0][0];
    expect(callArg).toContain('sort=status');
  });

  it('should normalize backup types correctly in response', async () => {
    const mockResponse = {
      data: {
        items: [
          {
            id: 'b1',
            name: 'manual-backup',
            type: 'manual',
            createdAt: '2026-05-11T20:00:00Z',
            fileCount: 5,
            sizeBytes: 1024,
          },
          {
            id: 'b2',
            name: 'auto-backup',
            type: 'auto',
            createdAt: '2026-05-11T21:00:00Z',
            fileCount: 3,
            sizeBytes: 2048,
          },
          {
            id: 'b3',
            name: 'pre-restore-backup',
            type: 'pre-restore',
            createdAt: '2026-05-11T22:00:00Z',
            fileCount: 2,
            sizeBytes: 512,
          },
        ],
        total: 3,
        page: 1,
        limit: 20,
      },
    };

    (api.default.get as any).mockResolvedValueOnce(mockResponse);

    const result = await backupService.listPaginated({ page: 1, limit: 20 });

    expect(result.items).toHaveLength(3);
    expect(result.items[0].type).toBe('manual');
    expect(result.items[1].type).toBe('auto');
    expect(result.items[2].type).toBe('pre-restore');
  });
});
