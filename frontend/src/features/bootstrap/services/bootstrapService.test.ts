import { describe, it, expect, vi, beforeEach } from 'vitest';
import { bootstrapService } from './bootstrapService';
// bootstrapService imports the default export from '@/shared/lib'
import * as apiLib from '@/shared/lib';
import type { ApiResponse, HostStatus } from '@/shared/types';

vi.mock('@/shared/lib', () => ({
  default: { get: vi.fn() },
  api: { get: vi.fn() },
}));

// bootstrapService uses `import api from '@/shared/lib'` (default)
const mockApi = (apiLib.default as unknown) as { get: ReturnType<typeof vi.fn> };

beforeEach(() => vi.clearAllMocks());

describe('bootstrapService', () => {
  describe('getStatus', () => {
    it('calls GET /bootstrap/status', async () => {
      const hostStatus: Partial<HostStatus> = {
        platform: 'linux',
        ready: true,
        interactive: false,
      };
      const apiResponse: ApiResponse<HostStatus> = {
        success: true,
        data: hostStatus as HostStatus,
        timestamp: new Date(0).toISOString(),
      };
      mockApi.get.mockResolvedValueOnce({ data: apiResponse });

      const result = await bootstrapService.getStatus();

      expect(mockApi.get).toHaveBeenCalledWith('/bootstrap/status');
      expect(result.data).toEqual(apiResponse);
    });

    it('propagates network errors', async () => {
      mockApi.get.mockRejectedValueOnce(new Error('network timeout'));

      await expect(bootstrapService.getStatus()).rejects.toThrow('network timeout');
    });

    it('calls the endpoint exactly once per call', async () => {
      mockApi.get.mockResolvedValue({ data: { success: true, data: {}, timestamp: '' } });

      await bootstrapService.getStatus();
      await bootstrapService.getStatus();

      expect(mockApi.get).toHaveBeenCalledTimes(2);
      expect(mockApi.get).toHaveBeenNthCalledWith(1, '/bootstrap/status');
      expect(mockApi.get).toHaveBeenNthCalledWith(2, '/bootstrap/status');
    });
  });
});
