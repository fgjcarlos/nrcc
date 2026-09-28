import { describe, it, expect, vi } from 'vitest';

// `useT` pulls react-i18next's I18nextProvider, which this environment
// cannot resolve (pre-existing repo limitation; CI pnpm install fixes
// it). Mirror the pattern used in other slice tests: mock the module
// so the test stays self-contained.
vi.mock('@/i18n', () => ({
  useT: () => ({
    t: (key: string) => key,
  }),
}));

import { render, screen } from '@testing-library/react';
import { ConfigurationFieldStatusRow } from './ConfigurationFieldStatusRow';
import type { FieldDiff } from '../hooks/useConfigurationDiff';

function diff(overrides: Partial<FieldDiff> = {}): FieldDiff {
  return {
    configuredValue: 1880,
    effectiveValue: 1880,
    formValue: 1880,
    effectiveLabel: 'Effective (NRCC-loaded)',
    source: 'settings.js',
    pending: false,
    restart: 'none',
    validation: { state: 'valid' },
    ...overrides,
  };
}

describe('<ConfigurationFieldStatusRow>', () => {
  it('renders the configured value, effective value, and source columns', () => {
    render(
      <table>
        <tbody>
          <ConfigurationFieldStatusRow fieldKey="uiPort" label="Port" diff={diff()} />
        </tbody>
      </table>
    );
    expect(screen.getByText('Port')).toBeInTheDocument();
    expect(screen.getByTestId('configuration-field-status-configured')).toHaveTextContent('1880');
    expect(screen.getByTestId('configuration-field-status-effective')).toHaveTextContent('1880');
    expect(screen.getByText('configuration:fieldStatus.source.settingsJs')).toBeInTheDocument();
  });

  it('renders the changed badge when pending=true', () => {
    render(
      <table>
        <tbody>
          <ConfigurationFieldStatusRow
            fieldKey="uiPort"
            label="Port"
            diff={diff({ pending: true, formValue: 2000, configuredValue: 1880, effectiveValue: 1880 })}
          />
        </tbody>
      </table>
    );
    expect(screen.getByTestId('configuration-field-status-pending')).toBeInTheDocument();
    expect(screen.getByTestId('configuration-field-status-configured')).toHaveTextContent('1880');
  });

  it('renders the restart chip with severity=hard when the field requires a hard restart', () => {
    render(
      <table>
        <tbody>
          <ConfigurationFieldStatusRow
            fieldKey="uiPort"
            label="Port"
            diff={diff({ pending: true, restart: 'hard' })}
          />
        </tbody>
      </table>
    );
    expect(screen.getByTestId('configuration-field-status-restart')).toHaveTextContent(/restart/i);
    expect(screen.getByTestId('configuration-field-status-restart')).toHaveAttribute('data-severity', 'hard');
  });

  it('renders an invalid validation chip when validation.state=invalid', () => {
    render(
      <table>
        <tbody>
          <ConfigurationFieldStatusRow
            fieldKey="authAdminPassword"
            label="Password"
            diff={diff({
                configuredValue: '',
                effectiveValue: '',
                formValue: 'short',
                source: 'form default',
                pending: true,
                validation: { state: 'invalid', message: 'Password must be at least 8 characters' },
              })}
          />
        </tbody>
      </table>
    );
    expect(screen.getByTestId('configuration-field-status-validation')).toHaveAttribute('data-state', 'invalid');
    expect(screen.getByText(/at least 8 characters/i)).toBeInTheDocument();
  });
});