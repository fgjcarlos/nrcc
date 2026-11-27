import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { i18n } from '@/i18n';
import enConfiguration from '@/locales/en/configuration.json';
import type { NodeRedConfigFormData } from '@/shared/types';
import { INITIAL_FORM_DATA } from '../lib/initialFormData';
import { SecuritySettings } from './SecuritySettings';

// Issue #1076 — the CORS card after the TLS card.
const copy = enConfiguration.httpNodeCors;

const renderSecurity = (overrides: Partial<NodeRedConfigFormData> = {}) => {
  const onUpdate = vi.fn();
  render(<SecuritySettings settings={{ ...INITIAL_FORM_DATA, ...overrides }} onUpdate={onUpdate} />);
  return onUpdate;
};

afterEach(async () => {
  await i18n.changeLanguage('en');
});

describe('SecuritySettings httpNodeCors card (issue #1076)', () => {
  it('starts off and reveals nothing until the toggle is on', async () => {
    const user = userEvent.setup();
    const onUpdate = renderSecurity();
    expect(screen.getByText(copy.title)).toBeVisible();
    expect(screen.getByRole('button', { name: new RegExp(copy.enable) })).toHaveAttribute('aria-pressed', 'false');
    expect(screen.queryByLabelText(new RegExp(copy.origins))).not.toBeInTheDocument();
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: new RegExp(copy.enable) }));
    expect(onUpdate).toHaveBeenCalledWith('httpNodeCorsEnabled', true);
  });

  it('shows the editable controls and the default method set when enabled', () => {
    renderSecurity({ httpNodeCorsEnabled: true });
    expect(screen.getByLabelText(new RegExp(copy.origins))).toHaveValue('');
    for (const method of ['GET', 'PUT', 'POST', 'DELETE']) {
      expect(screen.getByRole('checkbox', { name: method })).toBeChecked();
    }
    for (const method of ['HEAD', 'PATCH']) {
      expect(screen.getByRole('checkbox', { name: method })).not.toBeChecked();
    }
    expect(screen.getByRole('button', { name: new RegExp(copy.credentials) })).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByLabelText(new RegExp(copy.allowedHeaders))).toHaveValue('');
  });

  it('warns when the wildcard origin is used', () => {
    renderSecurity({ httpNodeCorsEnabled: true, httpNodeCorsOrigins: '*' });
    expect(screen.getByText(copy.wildcardWarning)).toBeVisible();
    expect(screen.queryByText(copy.wildcardCredentialsWarning)).not.toBeInTheDocument();
  });

  it('refuses the wildcard together with credentials', () => {
    renderSecurity({ httpNodeCorsEnabled: true, httpNodeCorsOrigins: '*', httpNodeCorsCredentials: true });
    expect(screen.getByText(copy.wildcardCredentialsWarning)).toBeVisible();
  });

  it('renders an unsupported block read-only with its raw source', () => {
    const raw = '  httpNodeCors: {\n    origin: (req, cb) => cb(null, "*"),\n  },';
    const onUpdate = renderSecurity({ httpNodeCorsUnsupported: true, httpNodeCorsRaw: raw });
    expect(screen.getByText(copy.unsupportedHelp)).toBeVisible();
    expect(screen.getByText(/origin: \(req, cb\)/)).toBeVisible();
    expect(screen.queryByLabelText(new RegExp(copy.origins))).not.toBeInTheDocument();
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: new RegExp(copy.enable) })).not.toBeInTheDocument();
    expect(onUpdate).not.toHaveBeenCalled();
  });

  it('keeps the existing TLS and credential controls alongside it', () => {
    renderSecurity({ httpNodeCorsEnabled: true });
    expect(screen.getByLabelText(/Credential Secret/)).toBeVisible();
    expect(screen.getByLabelText(/Private Key Path/)).toBeVisible();
    expect(screen.getByLabelText(/Certificate Path/)).toBeVisible();
  });
});
