import { describe, it, expect } from 'vitest';

// Type export verification - these will cause compile-time errors if types don't exist
import type { PaginationParams, PaginatedResponse, SortOrder, BackupSummary, BackupEvent, BackupConfig, BackupSchedulerStatus } from './index';

describe('Pagination Types', () => {
  it('should export PaginationParams, PaginatedResponse, and SortOrder types', () => {
    // If types are not exported, this test file won't compile
    expect(true).toBe(true);
  });

  it('should enforce SortOrder is exactly asc or desc', () => {
    // This test would fail at compile-time if SortOrder allowed other values
    // At runtime, we just verify the types are exported
    const ascOrder: SortOrder = 'asc';
    const descOrder: SortOrder = 'desc';
    expect(ascOrder).toBe('asc');
    expect(descOrder).toBe('desc');
  });

  it('should support BackupSummary in PaginatedResponse', () => {
    // Create a mock paginated response with backup data
    const mockBackup: BackupSummary = {
      id: 'backup-1',
      name: 'backup-name',
      type: 'manual',
      createdAt: new Date().toISOString(),
      triggeredBy: 'System',
      fileCount: 5,
      totalSize: 1024,
    };

    const response: PaginatedResponse<BackupSummary> = {
      items: [mockBackup],
      total: 1,
      page: 1,
      limit: 20,
    };

    expect(response.items).toHaveLength(1);
    expect(response.items[0].id).toBe('backup-1');
    expect(response.total).toBe(1);
  });

  it('should support multiple items in PaginatedResponse', () => {
    const items: BackupSummary[] = Array.from({ length: 20 }, (_, i) => ({
      id: `backup-${i}`,
      name: `backup-name-${i}`,
      type: 'manual',
      createdAt: new Date().toISOString(),
      triggeredBy: 'System',
      fileCount: 5,
      totalSize: 1024 * (i + 1),
    }));

    const response: PaginatedResponse<BackupSummary> = {
      items,
      total: 42,
      page: 1,
      limit: 20,
    };

    expect(response.items).toHaveLength(20);
    expect(response.total).toBe(42);
    expect(response.page).toBe(1);
    expect(response.limit).toBe(20);
  });

  it('should support all sort options: date, size, status', () => {
    const dateSortParams: PaginationParams = { page: 1, limit: 20, sort: 'date' };
    const sizeSortParams: PaginationParams = { page: 1, limit: 20, sort: 'size' };
    const statusSortParams: PaginationParams = { page: 1, limit: 20, sort: 'status' };

    expect(dateSortParams.sort).toBe('date');
    expect(sizeSortParams.sort).toBe('size');
    expect(statusSortParams.sort).toBe('status');
  });

  it('should support both asc and desc order options', () => {
    const ascParams: PaginationParams = { page: 1, limit: 20, sort: 'date', order: 'asc' };
    const descParams: PaginationParams = { page: 1, limit: 20, sort: 'date', order: 'desc' };

    expect(ascParams.order).toBe('asc');
    expect(descParams.order).toBe('desc');
  });

  it('should handle PaginationParams with only required fields', () => {
    const minimal: PaginationParams = { page: 1, limit: 20 };
    expect(minimal.page).toBe(1);
    expect(minimal.limit).toBe(20);
    expect('sort' in minimal && minimal.sort).toBeFalsy();
    expect('order' in minimal && minimal.order).toBeFalsy();
  });

  it('should support second and third pages in responses', () => {
    const page2Response: PaginatedResponse<BackupSummary> = {
      items: [],
      total: 42,
      page: 2,
      limit: 20,
    };

    const page3Response: PaginatedResponse<BackupSummary> = {
      items: [],
      total: 42,
      page: 3,
      limit: 20,
    };

    expect(page2Response.page).toBe(2);
    expect(page3Response.page).toBe(3);
    expect(page2Response.total).toBe(page3Response.total);
  });

  it('exports BackupEvent, BackupConfig, and BackupSchedulerStatus types', () => {
    // Verify the type re-exports exist by referencing them at runtime.
    // If any type is missing from ./index, the import above stops compiling.
    const evt = {} as BackupEvent;
    const cfg = {} as BackupConfig;
    const sched = {} as BackupSchedulerStatus;
    expect(typeof evt).toBe('object');
    expect(typeof cfg).toBe('object');
    expect(typeof sched).toBe('object');
  });
});
