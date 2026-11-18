import { describe, expect, it } from 'vitest';
import type { ReactNode } from 'react';
import { act, renderHook } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { http, HttpResponse } from 'msw';
import { server } from '@/test/msw/server';
import { backupConfig } from '@/test/msw/fixtures';
import { useBackupsActions } from './useBackupsActions';

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider
    client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}
  >
    {children}
  </QueryClientProvider>
);

describe('useBackupsActions retentionMutation', () => {
  it('persists manual, auto and pre-restore retention through POST /api/backups/config', async () => {
    const bodies: unknown[] = [];
    server.use(
      http.post('/api/backups/config', async ({ request }) => {
        const body = await request.json();
        bodies.push(body);
        return HttpResponse.json({ success: true, data: body });
      }),
    );

    const { result } = renderHook(() => useBackupsActions(), { wrapper });
    const config = { ...backupConfig, retentionManual: 7, retentionAuto: 14, retentionPreRestore: 3 };

    await act(async () => {
      await result.current.retentionMutation.mutateAsync(config);
    });

    expect(bodies).toEqual([
      expect.objectContaining({ retentionManual: 7, retentionAuto: 14, retentionPreRestore: 3 }),
    ]);
  });
});
