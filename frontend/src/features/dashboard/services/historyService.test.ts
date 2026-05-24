import { beforeEach, describe, expect, it, vi } from 'vitest';
import api from '@/shared/lib';
import { historyService } from './historyService';

vi.mock('@/shared/lib', () => ({
  default: {
    get: vi.fn(),
  },
}));

const mockGet = vi.mocked(api.get);

describe('historyService.getSystemHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls /system/history with default n=120', () => {
    mockGet.mockResolvedValueOnce({ data: { success: true, data: [], timestamp: '' } });

    historyService.getSystemHistory();

    expect(mockGet).toHaveBeenCalledWith('/system/history', { params: { n: 120 } });
  });

  it('calls /system/history with custom n when specified', () => {
    mockGet.mockResolvedValueOnce({ data: { success: true, data: [], timestamp: '' } });

    historyService.getSystemHistory(50);

    expect(mockGet).toHaveBeenCalledWith('/system/history', { params: { n: 50 } });
  });
});

describe('historyService.getRuntimeHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls /runtime/history with default n=50', () => {
    mockGet.mockResolvedValueOnce({ data: { success: true, data: { events: [], status: 'running' }, timestamp: '' } });

    historyService.getRuntimeHistory();

    expect(mockGet).toHaveBeenCalledWith('/runtime/history', { params: { n: 50 } });
  });

  it('calls /runtime/history with custom n when specified', () => {
    mockGet.mockResolvedValueOnce({ data: { success: true, data: { events: [], status: 'running' }, timestamp: '' } });

    historyService.getRuntimeHistory(30);

    expect(mockGet).toHaveBeenCalledWith('/runtime/history', { params: { n: 30 } });
  });
});
