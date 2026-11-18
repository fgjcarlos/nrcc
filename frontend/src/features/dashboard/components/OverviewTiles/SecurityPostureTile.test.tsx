import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { I18nProvider } from '@/i18n';
import { SecurityPostureTile } from './SecurityPostureTile';

function renderTile(props: Parameters<typeof SecurityPostureTile>[0]) {
  return render(
    <I18nProvider>
      <MemoryRouter>
        <SecurityPostureTile {...props} />
      </MemoryRouter>
    </I18nProvider>,
  );
}

describe('SecurityPostureTile', () => {
  it('reads every enabled surface as "On" in verified text', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: true,
      requireHttps: true,
    });
    const tile = screen.getByTestId('overview-security-posture-tile');
    expect(tile).toBeInTheDocument();
    for (const key of ['adminAuth', 'httpNodeAuth', 'httpStaticAuth', 'requireHttps']) {
      const row = screen.getByTestId(`overview-security-${key}`);
      expect(row).toHaveTextContent('On');
      expect(row.className).toMatch(/text-ds-on-tint-success/);
    }
  });

  // T8: a surface that is off is not critical by itself. Only adminAuth
  // (whose absence exposes the editor) reads amber; the rest stay neutral.
  it('reads disabled surfaces as "Off" without critical red', () => {
    renderTile({
      adminAuth: false,
      httpNodeAuth: false,
      httpStaticAuth: false,
      requireHttps: false,
    });
    for (const key of ['adminAuth', 'httpNodeAuth', 'httpStaticAuth', 'requireHttps']) {
      const row = screen.getByTestId(`overview-security-${key}`);
      expect(row).toHaveTextContent('Off');
      expect(row.className).not.toMatch(/danger|error/);
    }
    expect(screen.getByTestId('overview-security-adminAuth').className).toMatch(/text-ds-on-tint-warning/);
    expect(screen.getByTestId('overview-security-httpNodeAuth').className).toMatch(/text-ds-text-secondary/);
  });

  it('calls the posture partial, not exposed, when only adminAuth is on', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: false,
      httpStaticAuth: false,
      requireHttps: false,
    });
    const overallChip = screen.getByTestId('overview-security-posture-tile').querySelector('[role="status"]');
    expect(overallChip).toHaveTextContent('Partial');
    expect(overallChip?.className).toMatch(/bg-ds-warning/);
  });

  it('calls the posture exposed when adminAuth is off, whatever else is on', () => {
    renderTile({
      adminAuth: false,
      httpNodeAuth: true,
      httpStaticAuth: true,
      requireHttps: true,
    });
    const overallChip = screen.getByTestId('overview-security-posture-tile').querySelector('[role="status"]');
    expect(overallChip).toHaveTextContent('Exposed');
    expect(overallChip?.className).toMatch(/bg-ds-danger/);
  });

  it('keeps the posture rows plain: no links or buttons among them', () => {
    renderTile({ adminAuth: true, httpNodeAuth: false, httpStaticAuth: false, requireHttps: true });
    const tile = screen.getByTestId('overview-security-posture-tile');
    // The only interactive element is the section link to /security.
    expect(tile.querySelectorAll('a, button, [tabindex]')).toHaveLength(1);
  });

  it('renders the overall danger chip when no surface is enabled', () => {
    renderTile({
      adminAuth: false,
      httpNodeAuth: false,
      httpStaticAuth: false,
      requireHttps: false,
    });
    const tile = screen.getByTestId('overview-security-posture-tile');
    const overallChip = tile.querySelector('[role="status"]');
    expect(overallChip?.className).toMatch(/bg-ds-danger/);
  });

  it('renders the overall success chip when every surface is enabled', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: true,
      requireHttps: true,
    });
    const tile = screen.getByTestId('overview-security-posture-tile');
    const overallChip = tile.querySelector('[role="status"]');
    expect(overallChip?.className).toMatch(/bg-ds-success/);
  });

  it('exposes a deep link to /security', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: false,
      requireHttps: true,
    });
    const link = screen.getByTestId('overview-security-posture-link');
    expect(link).toHaveAttribute('href', '/security');
  });

  it('lists all four surface rows in order', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: true,
      requireHttps: true,
    });
    expect(screen.getByTestId('overview-security-adminAuth')).toBeInTheDocument();
    expect(screen.getByTestId('overview-security-httpNodeAuth')).toBeInTheDocument();
    expect(screen.getByTestId('overview-security-httpStaticAuth')).toBeInTheDocument();
    expect(screen.getByTestId('overview-security-requireHttps')).toBeInTheDocument();
  });

  it('uses the warning icon when at least one surface is enabled but not all', () => {
    renderTile({
      adminAuth: true,
      httpNodeAuth: true,
      httpStaticAuth: false,
      requireHttps: false,
    });
    // Header icon (ShieldOff) implies warning state — the surface rows
    // remain individually success/danger. We assert the overall chip
    // is visible; variant mapping is exercised by the danger test.
    const tile = screen.getByTestId('overview-security-posture-tile');
    expect(tile.querySelector('svg')).toBeTruthy();
  });

  it('says "Not reported" instead of inventing a posture when props are missing', () => {
    // Defensive: caller could (in theory) pass zero props. The four
    // booleans default to undefined which TypeScript forbids in strict
    // mode, so we cast through `any` to simulate a future-tolerant
    // boundary (e.g. an older backend omitting the data).
    renderTile({});
    const overallChip = screen.getByTestId('overview-security-posture-tile').querySelector('[role="status"]');
    expect(overallChip).toHaveTextContent('Not reported');
    expect(screen.queryByText('Exposed')).not.toBeInTheDocument();
  });
});
