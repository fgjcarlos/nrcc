import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { I18nProvider } from '@/i18n';
import { CompatModeChip } from './CompatModeChip';

function renderChip(props: Parameters<typeof CompatModeChip>[0]) {
  return render(
    <I18nProvider>
      <CompatModeChip {...props} />
    </I18nProvider>,
  );
}

describe('CompatModeChip', () => {
  it('renders the success palette when editable', () => {
    renderChip({ editable: true });
    const chip = screen.getByTestId('compat-mode-chip');
    expect(chip.className).toMatch(/bg-ds-success/);
  });

  it('renders the warning palette when read-only with a known 4.x runtime', () => {
    renderChip({ editable: false, mode: 'read-only', runtimeVersion: '4.0.9' });
    const chip = screen.getByTestId('compat-mode-chip');
    expect(chip.className).toMatch(/bg-ds-warning/);
  });

  it('renders the danger palette when read-only with an unknown runtime', () => {
    renderChip({ editable: false, mode: 'read-only', runtimeVersion: 'unknown' });
    const chip = screen.getByTestId('compat-mode-chip');
    expect(chip.className).toMatch(/bg-ds-danger/);
  });

  it('renders the neutral palette while editable is undefined (loading)', () => {
    renderChip({});
    const chip = screen.getByTestId('compat-mode-chip');
    expect(chip.className).toMatch(/bg-base-300/);
  });

  it('honours mode="editable" even when editable=false is passed through', () => {
    // The backend uses mode as the canonical flag; editable is a derived
    // boolean. When mode says editable the chip must say editable.
    renderChip({ editable: false, mode: 'editable' });
    const chip = screen.getByTestId('compat-mode-chip');
    expect(chip.className).toMatch(/bg-ds-success/);
  });

  it('includes the reason in the aria-label when read-only', () => {
    renderChip({ editable: false, mode: 'read-only', reason: 'Node-RED 4 detection' });
    const chip = screen.getByTestId('compat-mode-chip');
    expect(chip.getAttribute('aria-label')).toMatch(/Node-RED 4 detection/);
  });

  it('falls back to a generic aria-label when read-only without a reason', () => {
    renderChip({ editable: false, mode: 'read-only' });
    const chip = screen.getByTestId('compat-mode-chip');
    expect(chip.getAttribute('aria-label')).toMatch(/read-only/i);
  });
});
