/**
 * Deprecated. UI_COPY is now a forwarding stub; the real catalog tests
 * live in the i18n foundation test suite and the per-feature tests.
 *
 * @deprecated removed in slice 3 of issue #767
 */
import { describe, it, expect } from 'vitest';
import { i18n } from '@/i18n';

describe('UI_COPY legacy stub (slice 2 of #767)', () => {
  it('forwards cancel to the common catalog', () => {
    expect(i18n.t('common:cancel')).toBeTruthy();
  });
  it('forwards auth:login.adminNotConfigured to the auth catalog', () => {
    expect(i18n.t('auth:login.adminNotConfigured')).toBeTruthy();
  });
});
