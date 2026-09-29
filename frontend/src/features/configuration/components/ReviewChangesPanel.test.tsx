import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/react';
import { ReviewChangesPanel } from './ReviewChangesPanel';
import type {
  ConfigurationDiff,
  FieldDiff,
} from '../hooks/useConfigurationDiff';
import type { NodeRedConfigFormData } from '@/shared/types';

// Mock the i18n module so the component can resolve `useT` in this
// sandbox (the real `react-i18next` chain is unavailable locally).
vi.mock('@/i18n', () => ({
  useT: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (!params) return key;
      return key.replace(/\{\{(\w+)\}\}/g, (_, name) => String(params[name]));
    },
  }),
}));

function makeDiff(overrides: Partial<Record<keyof NodeRedConfigFormData, Partial<FieldDiff>>> = {}): ConfigurationDiff {
  const base = {
    configuredValue: 'old',
    effectiveValue: 'old',
    formValue: 'old',
    effectiveLabel: 'Effective (NRCC-loaded)',
    source: 'settings.js' as const,
    pending: false,
    restart: 'none' as const,
    validation: { state: 'valid' as const },
  };
  const out = {} as ConfigurationDiff;
  const keys: Array<keyof NodeRedConfigFormData> = [
    'uiPort', 'loggingConsoleLevel', 'editorPageTitle',
  ];
  for (const key of keys) {
    out[key] = { ...base, ...(overrides[key] ?? {}) } as FieldDiff;
  }
  return out;
}

describe('ReviewChangesPanel', () => {
  beforeEach(() => {
    // jsdom portal target.
    document.body.innerHTML = '';
  });
  afterEach(() => {
    cleanup();
  });

  it('renders nothing when closed', () => {
    render(
      <ReviewChangesPanel
        isOpen={false}
        fields={makeDiff()}
        pendingCount={0}
        target="settings.js"
        onApply={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(screen.queryByTestId('review-changes-panel')).toBeNull();
  });

  it('renders the empty state when there are no pending changes', () => {
    render(
      <ReviewChangesPanel
        isOpen
        fields={makeDiff()}
        pendingCount={0}
        target="settings.js"
        onApply={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByTestId('review-changes-panel')).toBeTruthy();
    expect(screen.getByTestId('review-changes-panel-empty')).toBeTruthy();
  });

  it('groups pending fields by restart severity in hard > soft > none order', () => {
    const fields = makeDiff({
      uiPort: { pending: true, restart: 'hard', formValue: '1880', configuredValue: '1881' },
      loggingConsoleLevel: { pending: true, restart: 'soft', formValue: 'info', configuredValue: 'error' },
      editorPageTitle: { pending: true, restart: 'none', formValue: 'New', configuredValue: 'Old' },
    });

    render(
      <ReviewChangesPanel
        isOpen
        fields={fields}
        pendingCount={3}
        target="settings.js"
        onApply={vi.fn()}
        onCancel={vi.fn()}
      />
    );

    const hard = screen.getByTestId('review-changes-panel-group-hard');
    const soft = screen.getByTestId('review-changes-panel-group-soft');
    const none = screen.getByTestId('review-changes-panel-group-none');
    expect(hard).toBeTruthy();
    expect(soft).toBeTruthy();
    expect(none).toBeTruthy();
    // Order: hard must come before soft, soft before none.
    const all = document.body.querySelectorAll('[data-testid^="review-changes-panel-group-"]');
    const ids = Array.from(all).map((el) => el.getAttribute('data-testid'));
    expect(ids).toEqual([
      'review-changes-panel-group-hard',
      'review-changes-panel-group-soft',
      'review-changes-panel-group-none',
    ]);
  });

  it('invokes onApply when the Apply button is clicked', () => {
    const onApply = vi.fn();
    const fields = makeDiff({
      uiPort: { pending: true, restart: 'hard', formValue: '1880' },
    });
    render(
      <ReviewChangesPanel
        isOpen
        fields={fields}
        pendingCount={1}
        target="settings.js"
        onApply={onApply}
        onCancel={vi.fn()}
      />
    );
    fireEvent.click(screen.getByTestId('review-changes-panel-apply'));
    expect(onApply).toHaveBeenCalledOnce();
  });

  it('invokes onCancel when Escape is pressed', () => {
    const onCancel = vi.fn();
    const fields = makeDiff({
      uiPort: { pending: true, restart: 'hard', formValue: '1880' },
    });
    render(
      <ReviewChangesPanel
        isOpen
        fields={fields}
        pendingCount={1}
        target="settings.js"
        onApply={vi.fn()}
        onCancel={onCancel}
      />
    );
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it('does not invoke onCancel when pending and Escape is disabled', () => {
    const onCancel = vi.fn();
    const fields = makeDiff({
      uiPort: { pending: true, restart: 'hard', formValue: '1880' },
    });
    render(
      <ReviewChangesPanel
        isOpen
        fields={fields}
        pendingCount={1}
        target="settings.js"
        isPending
        onApply={vi.fn()}
        onCancel={onCancel}
      />
    );
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onCancel).not.toHaveBeenCalled();
  });
});
