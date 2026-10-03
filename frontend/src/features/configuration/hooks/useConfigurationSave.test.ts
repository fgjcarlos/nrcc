/**
 * useConfigurationSave — issue #766 slice F W2.
 *
 * Tests the per-field save gate. The hook takes the diff (already
 * computed by `useConfigurationDiff`) and wraps the existing
 * `useConfigurationActions` mutation, gating the `mutateAsync` call on
 * `diff.canSave`. When the diff says no (no pending changes,
 * validation errors, or backend read-only) the hook short-circuits
 * the call and returns a structured reason.
 *
 * The aggregate `validateAuthFields` helper that used to live inside
 * `useConfigurationActions` is removed; per-field validation is the
 * new source of truth (see `validateConfigurationField`).
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

import type { ConfigurationDiffSummary } from './useConfigurationDiff';
import type { NodeRedConfigFormData } from '@/shared/types';

const mocks = vi.hoisted(() => ({
  mutateAsync: vi.fn(),
}));

vi.mock('./useConfigurationActions', () => ({
  useConfigurationActions: () => ({
    saveConfigMutation: { mutateAsync: mocks.mutateAsync, isPending: false },
    saveRawSettingsMutation: { mutateAsync: vi.fn(), isPending: false },
    handleSaveRawSettings: vi.fn(),
  }),
}));

vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { useConfigurationSave } from './useConfigurationSave';

function makeWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: any) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
}

function emptyForm(): NodeRedConfigFormData {
  return {
    uiPort: 1880,
    uiHost: '0.0.0.0',
    httpAdminRoot: '/',
    httpNodeRoot: '/',
    disableEditor: false,
    authEnabled: false,
    authAdminUser: '',
    authAdminPassword: '',
    authNodeHttpEnabled: false,
    authNodeHttpUser: '',
    authNodeHttpPassword: '',
    authStaticEnabled: false,
    authStaticUser: '',
    authStaticPassword: '',
    projectsEnabled: false,
  } as unknown as NodeRedConfigFormData;
}

function diffFixture(overrides: Partial<ConfigurationDiffSummary> = {}): ConfigurationDiffSummary {
  const pendingCount = overrides.pendingCount ?? 0;
  return {
    fields: {} as ConfigurationDiffSummary['fields'],
    pendingCount,
    restartRequired: 'none',
    validationErrors: {},
    canSave: pendingCount > 0,
    ...overrides,
  };
}

describe('useConfigurationSave', () => {
  beforeEach(() => {
    mocks.mutateAsync.mockReset();
    mocks.mutateAsync.mockResolvedValue(undefined);
  });

  it('calls mutateAsync when canSave is true and the form passes per-field validation', async () => {
    const { result } = renderHook(
      () => useConfigurationSave({ formData: emptyForm(), diff: diffFixture({ pendingCount: 1, canSave: true }) }),
      { wrapper: makeWrapper() },
    );

    await act(async () => {
      await result.current.save();
    });

    expect(mocks.mutateAsync).toHaveBeenCalledTimes(1);
  });

  it('short-circuits when there are no pending changes', async () => {
    const { result } = renderHook(
      () =>
        useConfigurationSave({
          formData: emptyForm(),
          diff: diffFixture({ canSave: false, pendingCount: 0 }),
        }),
      { wrapper: makeWrapper() },
    );

    const outcome = await result.current.save();

    expect(mocks.mutateAsync).not.toHaveBeenCalled();
    expect(outcome).toEqual({ ok: false, reason: 'no-pending-changes' });
  });

  it('short-circuits when there are validation errors', async () => {
    const { result } = renderHook(
      () =>
        useConfigurationSave({
          formData: emptyForm(),
          diff: diffFixture({
            pendingCount: 1,
            canSave: false,
            validationErrors: { uiPort: 'Port must be 1–65535' },
          }),
        }),
      { wrapper: makeWrapper() },
    );

    const outcome = await result.current.save();

    expect(mocks.mutateAsync).not.toHaveBeenCalled();
    expect(outcome).toEqual({ ok: false, reason: 'validation-errors', errors: { uiPort: 'Port must be 1–65535' } });
  });

  it('returns the mutation error when the save fails', async () => {
    mocks.mutateAsync.mockRejectedValueOnce(new Error('backend exploded'));

    const { result } = renderHook(
      () => useConfigurationSave({ formData: emptyForm(), diff: diffFixture({ pendingCount: 1, canSave: true }) }),
      { wrapper: makeWrapper() },
    );

    const outcome = await result.current.save();

    expect(outcome).toEqual({ ok: false, reason: 'mutation-failed', error: expect.any(Error) });
  });
});
