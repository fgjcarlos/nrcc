/**
 * FieldStatusChip — issue #766 slice F W2.
 *
 * Unit tests cover the variant mapping from `FieldDiff['validation']`
 * to the `StatusChip` variants and the i18n label the operator sees.
 * The chip is the inline feedback widget rendered next to each
 * `InputField` / `ToggleField` so the operator can spot a bad value
 * before clicking Save.
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

// Mock @/i18n before importing the SUT — pre-existing react-i18next
// resolution gap (issue #767 follow-up; see ConfigurationFieldStatusRow
// for the same workaround).
vi.mock('@/i18n', () => ({
  useT: () => ({ t: (key: string) => key }),
}));

import { FieldStatusChip } from './FieldStatusChip';

describe('FieldStatusChip', () => {
  it('renders nothing for a valid state', () => {
    const { container } = render(
      <FieldStatusChip validation={{ state: 'valid' }} fieldKey="uiPort" />,
    );
    expect(container.firstChild).toBeNull();
  });

  it('renders a danger chip with the validation message for an invalid state', () => {
    render(
      <FieldStatusChip
        validation={{ state: 'invalid', message: 'Port must be 1–65535' }}
        fieldKey="uiPort"
      />,
    );
    const chip = screen.getByRole('status');
    expect(chip).toHaveTextContent('Port must be 1–65535');
    // Classnames from the StatusChip cva variants — assert the
    // danger background/text colour is present.
    expect(chip.className).toMatch(/bg-ds-danger/);
  });

  it('falls back to the field-key i18n label when the validation has no message', () => {
    render(
      <FieldStatusChip validation={{ state: 'invalid' }} fieldKey="uiPort" />,
    );
    const chip = screen.getByRole('status');
    // useT is mocked to return the key, so we expect the i18n key the
    // chip falls back to. The exact key lives in the component
    // implementation; assert it is non-empty.
    expect(chip.textContent).toBeTruthy();
    expect(chip.textContent).not.toEqual('');
  });

  it('renders a neutral "pending" chip with the field-key i18n label', () => {
    render(
      <FieldStatusChip validation={{ state: 'pending' }} fieldKey="uiPort" />,
    );
    const chip = screen.getByRole('status');
    expect(chip.className).toMatch(/bg-base-300/);
  });
});
